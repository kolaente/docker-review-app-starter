# Review Proxy Design

An on-demand reverse proxy that lazily starts Docker Compose stacks when a web request arrives and tears them down after idle timeout. Designed for review/preview apps — CI pushes an image, reviewers visit a subdomain, the environment spins up automatically.

## Architecture Overview

Three components on a shared Docker network:

1. **Traefik** — TLS termination, wildcard cert for `*.review.example.com` via DNS-01 challenge, routes all traffic to the proxy.
2. **Review proxy** (Go binary) — lazy-start logic, idle cleanup, reverse-proxies to compose stacks.
3. **Compose stacks** — one per subdomain, started and stopped by the proxy.

Traefik is configured with a single catch-all rule: any request to `*.review.example.com` forwards to the review proxy container. The proxy listens on plain HTTP (`:80`) internally.

### Config

```yaml
domain: review.example.com
compose_template: docker-compose.template.yml
target_service: app
target_port: 8080
idle_timeout: 5m
```

No registry config — the proxy parses registry/repo from the compose template's image references. No TLS config — Traefik handles it.

## Request Flow

When a request arrives at the proxy for subdomain `pr-42`:

### 1. Check in-memory state

The proxy keeps a map of `subdomain -> stack state`. Possible states:

- `running` — stack is up, proxy the request.
- `starting` — stack is booting, serve "preparing" page.
- `not_found` — previously checked, image doesn't exist.
- unknown — never seen this subdomain before.

### 2. If running

- Reset the idle timer.
- If last image digest check was >5 min ago, query the registry API in the background. If digest changed, run `docker compose pull && docker compose up -d` (non-blocking, keep proxying to current containers).
- Reverse-proxy the request to `review-pr-42-<target_service>-1:<target_port>` on the shared network.

### 3. If unknown

- Parse the registry/repo from the template, query registry API for tag `pr-42`.
- Tag missing -> set state to `not_found`, serve error page.
- Tag exists -> set state to `starting`, start stack in background.
- Serve "preparing environment" page with `<meta http-equiv="refresh" content="3">`.

### 4. If starting

- Check if the container is reachable yet.
- Not ready -> serve "preparing" page again.
- Ready -> set state to `running`, proxy the request.

### 5. If not_found

- Serve error page. Re-check the registry on the next request if last check was >1 min ago (in case the image was pushed since).

## Image Update Check

When a request hits a running stack and the last digest check was >5 min ago:

1. Query the registry API for the current digest of the tag.
2. Compare to the digest stored when the stack was last started/updated.
3. If unchanged, update the last-check timestamp.
4. If changed:
   - Run `docker compose pull` then `docker compose up -d`.
   - Docker Compose handles the rolling replacement — old containers stay up until new ones are ready.
   - Update the stored digest and timestamp.
   - All of this happens in the background, requests keep proxying during the update.

Registry auth: the proxy container mounts the Docker config (e.g. `~/.docker/config.json`) or uses credential helpers. Same auth Docker uses for `docker pull`.

## Idle Cleanup

Each running stack has a timer that resets on every proxied request. When the timer fires (`idle_timeout`, default 5 min):

1. Run `docker compose -p review-pr-42 down --remove-orphans --volumes`.
2. Remove the subdomain from the in-memory map.

On the next request for that subdomain, it goes through the full flow again.

**Edge case:** a request arrives while the stack is being torn down. The proxy transitions the state to `starting` and queues a fresh start after the teardown completes, rather than trying to start and stop concurrently.

## Compose Stack Management

The proxy manages stacks using the `docker compose` CLI (shelling out).

### Starting a stack for subdomain `pr-42`

1. Copy the template to a temp directory.
2. Replace `${SUBDOMAIN}` with `pr-42` in the file.
3. Run `docker compose -p review-pr-42 -f <tempfile> up -d`.

The template itself includes the shared network:

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

The proxy resolves the container at `review-pr-42-app-1` on the `review-proxy` network.

### Stopping

```
docker compose -p review-pr-42 down --remove-orphans --volumes
```

### Proxy startup recovery

On startup, check for any existing `review-*` compose projects (via `docker compose ls`). Re-adopt them into the in-memory map so the proxy can resume managing stacks that survived a proxy restart.

## Deployment

Ships as a `docker-compose.yml`:

```yaml
services:
  traefik:
    image: traefik:v3
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - traefik-certs:/acme
    environment:
      - CF_API_TOKEN=${CF_API_TOKEN}
    # traefik static config via command args or mounted file

  proxy:
    build: .
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./config.yaml:/etc/review-proxy/config.yaml
      - ./docker-compose.template.yml:/etc/review-proxy/template.yml
    networks:
      - review-proxy
    labels:
      - "traefik.http.routers.review.rule=HostRegexp(`{subdomain:.+}.review.example.com`)"
      - "traefik.http.routers.review.tls=true"
      - "traefik.http.routers.review.tls.certresolver=letsencrypt"
      - "traefik.http.routers.review.tls.domains[0].main=review.example.com"
      - "traefik.http.routers.review.tls.domains[0].sans=*.review.example.com"

networks:
  review-proxy:
    name: review-proxy

volumes:
  traefik-certs:
```

Users clone the repo, add their `docker-compose.template.yml`, set env vars for the DNS provider, and run `docker compose up -d`.
