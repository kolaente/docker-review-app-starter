---
# docker-review-app-starter-umxb
title: Create README with setup and usage instructions
status: completed
type: task
priority: normal
created_at: 2026-03-03T14:03:31Z
updated_at: 2026-03-03T14:23:11Z
parent: docker-review-app-starter-014k
blocked_by:
    - docker-review-app-starter-l7fg
---

## Create README

Write a README.md covering what the project does, how to set it up, and how to use it.

### Files
- Create: `README.md`

### Step 1: Write README.md

Create `README.md` with the following sections:

- **What it does** — one-paragraph summary: an on-demand reverse proxy that lazily starts Docker Compose stacks when a web request arrives and tears them down after idle timeout. Designed for review/preview apps.
- **How it works** — brief explanation of the request flow: subdomain extraction → registry check → compose stack start → reverse proxy. Mention the "preparing" page with auto-refresh, the "not found" page, idle cleanup, and automatic image update detection.
- **Prerequisites** — Docker, Docker Compose, a container registry, a domain with DNS pointing to the server, DNS provider API credentials for Let's Encrypt wildcard certs.
- **Setup** — step by step:
  1. Clone the repo
  2. Copy `config.example.yaml` to `config.yaml` and fill in values
  3. Copy `docker-compose.template.example.yml` to `docker-compose.template.yml` and customize
  4. Copy `.env.example` to `.env` and set DNS provider credentials
  5. Run `docker compose up -d`
- **Configuration reference** — table of config.yaml fields: `domain`, `compose_template`, `target_service`, `target_port`, `idle_timeout`
- **Compose template** — explain the `${SUBDOMAIN}` placeholder and the requirement to join the `review-proxy` external network
- **How CI integrates** — just build and push images tagged with the branch/PR identifier. No webhook or API call needed. When someone visits the subdomain, the proxy handles the rest.

### Step 2: Commit

```bash
git add README.md
git commit -m "docs: add README with setup and usage instructions"
```
