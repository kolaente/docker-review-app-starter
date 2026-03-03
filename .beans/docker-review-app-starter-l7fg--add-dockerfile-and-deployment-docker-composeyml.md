---
# docker-review-app-starter-l7fg
title: Add Dockerfile and deployment docker-compose.yml
status: todo
type: task
priority: normal
created_at: 2026-03-03T14:02:11Z
updated_at: 2026-03-03T14:04:16Z
parent: docker-review-app-starter-014k
blocked_by:
    - docker-review-app-starter-er0o
---

## Add Dockerfile and deployment docker-compose.yml

Package the proxy as a Docker image and provide the deployment compose file with Traefik.

### Files
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Create: `docker-compose.template.example.yml`

### Step 1: Create Dockerfile

Create `Dockerfile`:

```dockerfile
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o review-proxy .

FROM alpine:3.21
RUN apk add --no-cache docker-cli docker-cli-compose
COPY --from=builder /app/review-proxy /usr/local/bin/review-proxy
ENTRYPOINT ["review-proxy"]
```

Note: The runtime image needs `docker-cli` and `docker-cli-compose` because the proxy shells out to `docker compose`.

### Step 2: Create deployment docker-compose.yml

Create `docker-compose.yml`:

```yaml
services:
  traefik:
    image: traefik:v3.3
    command:
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.dnschallenge=true"
      - "--certificatesresolvers.letsencrypt.acme.dnschallenge.provider=${DNS_PROVIDER:-cloudflare}"
      - "--certificatesresolvers.letsencrypt.acme.storage=/acme/acme.json"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - traefik-certs:/acme
    env_file:
      - .env
    networks:
      - review-proxy
    restart: unless-stopped

  proxy:
    build: .
    command: ["-config", "/etc/review-proxy/config.yaml"]
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./config.yaml:/etc/review-proxy/config.yaml:ro
      - ./docker-compose.template.yml:/etc/review-proxy/template.yml:ro
    networks:
      - review-proxy
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.review.rule=HostRegexp(`.+\\.${DOMAIN}`)"
      - "traefik.http.routers.review.entrypoints=websecure"
      - "traefik.http.routers.review.tls=true"
      - "traefik.http.routers.review.tls.certresolver=letsencrypt"
      - "traefik.http.routers.review.tls.domains[0].main=${DOMAIN}"
      - "traefik.http.routers.review.tls.domains[0].sans=*.${DOMAIN}"
      - "traefik.http.services.review.loadbalancer.server.port=80"
      # HTTP to HTTPS redirect
      - "traefik.http.routers.review-http.rule=HostRegexp(`.+\\.${DOMAIN}`)"
      - "traefik.http.routers.review-http.entrypoints=web"
      - "traefik.http.middlewares.review-redirect.redirectscheme.scheme=https"
      - "traefik.http.routers.review-http.middlewares=review-redirect"
    restart: unless-stopped

networks:
  review-proxy:
    name: review-proxy

volumes:
  traefik-certs:
```

### Step 3: Create example compose template

Create `docker-compose.template.example.yml`:

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

### Step 4: Create .env.example

Create `.env.example`:

```
# DNS provider for Let's Encrypt wildcard certs
DNS_PROVIDER=cloudflare
CF_API_TOKEN=your-cloudflare-api-token

# Domain for review apps
DOMAIN=review.example.com
```

### Step 5: Create .gitignore

Create `.gitignore`:

```
review-proxy
config.yaml
docker-compose.template.yml
.env
```

### Step 6: Verify Docker build

Run: `docker build -t review-proxy .`
Expected: Builds successfully.

### Step 7: Commit

```bash
git add Dockerfile docker-compose.yml docker-compose.template.example.yml .env.example .gitignore
git commit -m "feat: add Dockerfile and deployment compose with Traefik"
```
