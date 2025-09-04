# docker-credential-calieph

A Docker credential that provides short-lived (ephemeral) credentials for multiple container registries.
It is designed for CI/CD pipelines and developer workflows where it is best not to store long-lived credentials on disk.

## Supported Registries

### Docker Hub

- Uses one of usernanme/password based on various possible environment variables.
  - username: `DOCKERHUB_USERNAME`, `DOCKER_USERNAME`, `DOCKERHUB_USER`, `DOCKER_USER`
  - password: `DOCKERHUB_PASSWORD`, `DOCKER_PASSWORD`, `DOCKERHUB_TOKEN`, `DOCKER_TOKEN`
- Falls back to anonymous access if no credentials are provided.

### Quay.io

- Uses one of usernanme/password based on various possible environment variables.
  - username: `QUAY_USERNAME`, `QUAY_USER`
  - password: `QUAY_PASSWORD`, `QUAY_TOKEN`
- Falls back to anonymous access using empty credentials.

### Google Container Registry (GCR) and Google Artifact Registry (GAR)

- Uses Google Application Default Credentials (ADC) to generate short-lived access tokens.

## Usage

### As credential helper

Configure Docker to use the credential helper by adding the following to your `~/.docker/config.json`:

```json
{
  "credHelpers": {
    "asia.gcr.io": "calieph",
    "eu.gcr.io": "calieph",
    "gcr.io": "calieph",
    "marketplace.gcr.io": "calieph",
    "staging-k8s.gcr.io": "calieph",
    "us.gcr.io": "calieph",
    "europe-west3-docker.pkg.dev": "calieph"
  }
}
```

Run Docker commands as normal and the credential helper will be used automatically for all supported specified registries.
Be sure to set any required environment variables for the registries you want to access.

### As credential store

Configure Docker to use the credential helper by adding the following to your `~/.docker/config.json`:

```json
{
  "credsStore": "calieph"
}
```

Run Docker commands as normal using `go-build` image and the credential helper will be used automatically.
Be sure to mount or set any required environment variables for the registries you want to access.

For example:

- to use Docker Hub credentials stored in environment variables:

  ```sh
  docker run -t --entrypoint /bin/sh \
    -v $HOME/.docker/config.json:/root/.docker/config.json \
    -e DOCKERHUB_USERNAME -e DOCKERHUB_PASSWORD \
    calico/go-build:<GOBUILD_VERSION> -c "crane <do-something>"
  ```

- to use GCR/GAR credentials from a service account JSON file:

  ```sh
  docker run -t --entrypoint /bin/sh \
    -v $HOME/.docker/config.json:/root/.docker/config.json \
    -v $HOME/gcp-key.json:/gcp-key.json:ro \
    -e GOOGLE_APPLICATION_CREDENTIALS=/gcp-key.json \
    calico/go-build:<GOBUILD_VERSION> -c "crane <do-something>"
  ```

- to use Quay.io credentials stored in environment variables:

  ```sh
  docker run -t --entrypoint /bin/sh \
    -v $HOME/.docker/config.json:/root/.docker/config.json \
    -e QUAY_USER -e QUAY_TOKEN \
    calico/go-build:<GOBUILD_VERSION> -c "crane <do-something>"
  ```

- to use GCR/GAR credentials from a service account JSON file and Docker Hub credentials from environment variables:

  ```sh
  docker run -t --entrypoint /bin/sh \
    -v $HOME/.docker/config.json:/root/.docker/config.json \
    -v $HOME/gcp-key.json:/gcp-key.json:ro \
    -e GOOGLE_APPLICATION_CREDENTIALS=/gcp-key.json \
    -e DOCKER_USER -e DOCKER_TOKEN \
    calico/go-build:<GOBUILD_VERSION> -c "crane <do-something>"
  ```
