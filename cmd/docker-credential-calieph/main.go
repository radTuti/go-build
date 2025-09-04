// Copyright (c) 2025 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2/google"
)

const (
	// Name of the credential helper. This is updated during the build.
	Name = "docker-credential-calieph"

	// Docker username when authenticating using OAuth2 token
	dockerOAuth2User = "oauth2accesstoken"
)

// Filled during the build process.
var (
	// Version of the credential helper.
	Version = "0.0.0+unknown"

	// Revision of the credential helper.
	Revision = "unknown"
)

type credential struct {
	ServerURL string
	Username  string
	Secret    string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stdout, usage())
		os.Exit(1)
	}
	action := os.Args[1]
	switch action {
	case "get":
		if err := get(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	case "help", "--help", "-h":
		fmt.Fprintln(os.Stdout, usage())
		os.Exit(0)
	case "version", "--version", "-v":
		fmt.Fprintf(os.Stdout, "%s %s (rev: %s)\n", Name, Version, Revision)
		os.Exit(0)
	case "store", "erase", "list":
		fmt.Fprintf(os.Stderr, "(UNIMPLEMENTED) %s is ephermal and does not %s credentials\n", Name, action)
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stdout, usage())
		os.Exit(1)
	}
}

func usage() string {
	return fmt.Sprintf("Usage: %s [get|store|erase|list|version|help]", Name)
}

func get(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	buffer := new(bytes.Buffer)
	for scanner.Scan() {
		buffer.Write(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("failed to read input: %w", err)
	}

	serverURL := strings.TrimSpace(buffer.String())

	if serverURL == "" {
		serverURL = "https://index.docker.io/v1/"
	}

	ctx := context.Background()
	var cred credential
	switch {
	case isGoogleRegistry(serverURL):
		c, err := getGoogleCredentials(ctx)
		if err != nil {
			return fmt.Errorf("failed to get Google credentials: %w", err)
		}
		cred = c
	case isQuay(serverURL):
		c, err := getQuayCredentials()
		if err != nil {
			return fmt.Errorf("failed to get Quay credentials: %w", err)
		}
		cred = c
	case isDockerHub(serverURL):
		c, err := getDockerHubCredentials()
		if err != nil {
			return fmt.Errorf("failed to get Docker Hub credentials: %w", err)
		}
		cred = c
	}
	cred.ServerURL = serverURL

	buffer.Reset()
	if err := json.NewEncoder(buffer).Encode(cred); err != nil {
		return fmt.Errorf("failed to encode output: %w", err)
	}
	fmt.Fprint(out, buffer.String())
	return nil
}

func isGoogleRegistry(host string) bool {
	h := strings.ToLower(host)
	return strings.Contains(h, "gcr.io") || strings.Contains(h, "pkg.dev")
}

func getGoogleCredentials(ctx context.Context) (credential, error) {
	creds, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		return credential{}, fmt.Errorf("failed to find default credentials: %w", err)
	}
	token, err := creds.TokenSource.Token()
	if err != nil {
		return credential{}, fmt.Errorf("failed to get token: %w", err)
	}
	return credential{
		Username: dockerOAuth2User,
		Secret:   token.AccessToken,
	}, nil
}

func isQuay(host string) bool {
	return strings.Contains(strings.ToLower(host), "quay.io")
}

func getQuayCredentials() (credential, error) {
	username := getEnv("QUAY_USERNAME", "QUAY_USER")
	password := getEnv("QUAY_PASSWORD", "QUAY_TOKEN")
	if username != "" && password != "" {
		return credential{
			Username: username,
			Secret:   password,
		}, nil
	}

	// Quay allows access to public repositories without authentication
	// so we return an empty credential
	return credential{}, nil
}

func isDockerHub(host string) bool {
	return strings.Contains(strings.ToLower(host), "docker.io")
}

func getDockerHubCredentials() (credential, error) {
	username := getEnv("DOCKERHUB_USERNAME", "DOCKERHUB_USER", "DOCKER_USERNAME", "DOCKER_USER")
	password := getEnv("DOCKERHUB_PASSWORD", "DOCKERHUB_TOKEN", "DOCKER_PASSWORD", "DOCKER_TOKEN")
	if username != "" && password != "" {
		return credential{
			Username: username,
			Secret:   password,
		}, nil
	}

	// return anonymous credential
	req, err := http.NewRequest(http.MethodGet, "https://auth.docker.io/token?service=registry.docker.io&scope=repository:library/alpine:pull", nil)
	if err != nil {
		return credential{}, fmt.Errorf("failed to create request for token: %w", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return credential{}, fmt.Errorf("failed to get token: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return credential{}, fmt.Errorf("failed to get token: status code %d", res.StatusCode)
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return credential{}, fmt.Errorf("failed to decode token response: %w", err)
	}
	return credential{
		Username: dockerOAuth2User,
		Secret:   body.Token,
	}, nil
}

func getEnv(keys ...string) string {
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok && value != "" {
			return value
		}
	}
	return ""
}
