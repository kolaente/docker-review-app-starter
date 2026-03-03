---
# docker-review-app-starter-014k
title: Implement review proxy
status: todo
type: epic
created_at: 2026-03-03T13:58:49Z
updated_at: 2026-03-03T13:58:49Z
---

# Review Proxy Implementation

**Goal:** Build an on-demand reverse proxy (Go) that lazily starts Docker Compose stacks when a web request arrives and tears them down after idle timeout.

**Architecture:** A Go HTTP reverse proxy sits behind Traefik (TLS termination). On incoming requests, it extracts the subdomain, checks a Docker registry for the image tag, starts a compose stack from a template with the subdomain substituted in, and proxies traffic. Idle stacks are torn down after a configurable timeout. Image digests are periodically checked to pick up new pushes.

**Tech Stack:** Go, Docker Compose CLI, Docker Registry HTTP API v2, net/http/httputil.ReverseProxy, Traefik v3
