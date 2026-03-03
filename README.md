# docker-review-app-starter

## What it does

An on-demand reverse proxy that lazily starts Docker Compose stacks when a web request arrives and tears them down after an idle timeout. Designed for review/preview apps: CI pushes a Docker image tagged with the branch or PR identifier, a reviewer visits the corresponding subdomain, and the environment spins up automatically -- no webhook, no API call, no manual deployment step.

## How it works

The system consists of Traefik (TLS termination via wildcard Let's Encrypt certs), a Go-based review proxy, and per-subdomain Compose stacks, all on a shared Docker network.

When a request arrives for `pr-42.review.example.com`:

1. **Subdomain extraction** -- The proxy strips the base domain to identify the subdomain (`pr-42`).
2. **Registry check** -- The proxy queries the container registry for a tag matching that subdomain. If no image exists, it serves a "not found" page explaining that CI has not yet pushed an image for this branch.
3. **Stack start** -- If the image exists, the proxy renders the Compose template with the subdomain substituted in, starts the stack via `docker compose up`, and serves a "preparing environment" page that auto-refreshes every 3 seconds.
4. **Reverse proxy** -- Once the container is healthy, subsequent requests are proxied directly to the target service inside the stack.
5. **Idle cleanup** -- Each stack has an idle timer that resets on every proxied request. When the timeout expires, the stack is torn down with `docker compose down`. The next request starts the flow from the beginning.
6. **Automatic image updates** -- While a stack is running, the proxy periodically checks the registry for a new image digest. If the image has changed (e.g., a new CI push), it pulls the update and restarts the stack in the background without interrupting active requests.

## Prerequisites

- **Docker** and **Docker Compose** (v2, the `docker compose` plugin)
- A **container registry** (Docker Hub, GitHub Container Registry, GitLab Registry, etc.) accessible from the server
- A **domain** with a wildcard DNS record (`*.review.example.com`) pointing to the server
- **DNS provider API credentials** for Let's Encrypt DNS-01 challenge (used by Traefik to obtain wildcard TLS certificates). Cloudflare is preconfigured; see the [Traefik docs](https://doc.traefik.io/traefik/https/acme/#providers) for other providers.

## Setup

1. **Clone the repo**

   ```sh
   git clone https://github.com/kolaente/docker-review-app-starter.git
   cd docker-review-app-starter
   ```

2. **Create the config file**

   ```sh
   cp config.example.yaml config.yaml
   ```

   Edit `config.yaml` and set your domain, template path, target service name, target port, and idle timeout. See the [Configuration reference](#configuration-reference) below.

3. **Create the Compose template**

   ```sh
   cp docker-compose.template.example.yml docker-compose.template.yml
   ```

   Customize the template for your application. See [Compose template](#compose-template) below.

4. **Set environment variables**

   ```sh
   cp .env.example .env
   ```

   Edit `.env` and fill in your DNS provider credentials and domain:

   ```
   DNS_PROVIDER=cloudflare
   CF_API_TOKEN=your-actual-api-token
   DOMAIN=review.example.com
   ```

5. **Start the stack**

   ```sh
   docker compose up -d
   ```

   Traefik will automatically obtain a wildcard certificate for your domain. The review proxy is now listening for requests.

## Configuration reference

The file `config.yaml` controls the review proxy behavior.

| Field | Type | Description |
|---|---|---|
| `domain` | string | Base domain for review apps (e.g., `review.example.com`). Subdomains are extracted from incoming request hosts relative to this value. |
| `compose_template` | string | Path to the Docker Compose template file used to start stacks. Inside the proxy container this is mounted at `/etc/review-proxy/template.yml`. |
| `target_service` | string | Name of the service inside the Compose template to proxy requests to (e.g., `app`). |
| `target_port` | integer | Port the target service listens on inside its container (e.g., `8080`). |
| `idle_timeout` | duration | How long a stack can remain idle (no proxied requests) before it is torn down. Accepts Go duration strings such as `5m`, `30m`, or `1h`. |

Example:

```yaml
domain: review.example.com
compose_template: docker-compose.template.yml
target_service: app
target_port: 8080
idle_timeout: 5m
```

## Compose template

The Compose template is a standard `docker-compose.yml` file with one special placeholder: `${SUBDOMAIN}`. When a stack is started, the proxy replaces every occurrence of `${SUBDOMAIN}` with the actual subdomain (e.g., `pr-42`), writes the result to a temp file, and runs `docker compose up` against it.

Your template **must** join the `review-proxy` external network so that the proxy can reach the target service container:

```yaml
services:
  app:
    image: registry.example.com/myapp:${SUBDOMAIN}
    networks:
      - review-proxy

networks:
  review-proxy:
    external: true
```

The proxy resolves the container by its Compose-generated name on the shared network (`review-<subdomain>-<service>-1`), so no port publishing is needed.

You can add databases, workers, volumes, or any other services to the template. Just make sure the service specified by `target_service` in your config is on the `review-proxy` network.

## How CI integrates

No webhook or deployment API is required. Your CI pipeline only needs to:

1. Build the Docker image for the branch or pull request.
2. Push it to your registry, tagged with the branch name or PR identifier.

For example:

```sh
docker build -t registry.example.com/myapp:pr-42 .
docker push registry.example.com/myapp:pr-42
```

When someone visits `pr-42.review.example.com`, the proxy detects that the image exists, starts the stack, and begins proxying requests. If CI pushes a new version of the same tag, the proxy will detect the digest change and update the running stack automatically.
