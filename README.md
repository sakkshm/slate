# Slate

A developer-first Platform-as-a-Service (PaaS) that builds and deploys websites. Connect a GitHub repo once, after that, **every push is
built, stored, and deployed automatically** to `<slug>.slate.sakkshm.me`.

Slate is built around three ideas:

- **Push to deploy.** A GitHub webhook wakes Slate up, a worker builds the repo
  in an isolated Docker container, and the output goes live with HTTPS included.
- **Content-addressed artifacts.** Build output is stored by its SHA-256 hash,
  so identical builds are deduplicated and every deployment is immutable and
  reproducible.
- **In-process gateway.** Deployed sites are served by the API itself through a
  host-based gateway that pulls artifacts from object storage on demand. No
  separate CDN or proxy tier to run.

## Features

- **Deploy on every push** — connect a GitHub repo, and each push to your prod
  branch triggers a build. No CI config to write.
- **Isolated builds** — each build runs in its own container with hard memory,
  CPU, and process limits, a read-only rootfs, no network access to other
  containers, and all Linux capabilities dropped.
- **Live build logs** — logs stream into the browser over SSE in real time, with
  durable replay for late-joining viewers.
- **Environment variables** — per-project secrets, encrypted at rest with
  AES-256-GCM and injected into the build container at build time.
- **GitHub commit status** — the worker reports `pending`/`success`/`failure`
  back to GitHub for every build.
- **Content-addressed gateway** — deployments are served by SHA-256 hash through
  an in-process cache, with a Postgres + object storage fallback.
- **Custom subdomains** — every deployment is live instantly at
  `https://<slug>.slate.sakkshm.me`, with reserved labels like `api` and `www`
  excluded.
- **Self-hosted** — one `docker compose` command brings up everything.

## Architecture at a glance

```
Git push ──▶ GitHub webhook ──▶ API (control plane) ──▶ Redis stream
                                                               │
                    browser ◀── Gateway ◀── MinIO ◀── worker ──┘
```

- **API** (`backend/cmd/api`) — the control plane: GitHub OAuth, project and
  build CRUD, encrypted env vars, the webhook receiver, SSE build logs, and the
  in-process deployment gateway.
- **Worker** (`backend/cmd/worker`) — consumes build jobs from a Redis stream,
  runs each build in an isolated Docker container, streams logs, uploads the
  output to MinIO, and publishes a deployment pointer.
- **Frontend** (`frontend`) — React + TypeScript dashboard: public landing page,
  project wizard, live build logs, deployment history, and project settings.

See [`docs/architecture.md`](docs/architecture.md) for the full design.

## Requirements

- Docker with Docker Compose (for Postgres, Redis, MinIO, and the backend)
- pnpm (for the frontend)
- A GitHub App for OAuth, webhooks, and repo access (see
  [`docs/deploy.md`](docs/deploy.md))

## Quickstart (development)

```bash
# 1. Start the backend stack (API, worker, db, redis, minio, base-runner)
cd backend
cp .env.example .env        # fill in GitHub App credentials (see docs/deploy.md)
docker compose --profile dev up --build

# 2. Start the frontend in another terminal
cd frontend
pnpm install
pnpm dev
```

- Frontend: http://localhost:5173
- API: http://localhost:8080 (health check: `GET /health`)
- MinIO console: http://localhost:9001

The dev profile hot-reloads the backend with `air`, rebuilds the base runner
image used for builds, and serves the API with the gateway enabled.

## Production

```bash
cd backend
docker compose --profile prod up -d --build
```

The `prod` profile runs hardened multi-stage images: the API as a non-root user
with no Docker socket, and the worker with the socket it needs to launch build
containers. See [`docs/deploy.md`](docs/deploy.md) for DNS + SSL setup, GitHub
App configuration, and the security hardening guide (rate limits, CORS, trusted
proxies).

## Configuration

Copy `backend/.env.example` to `backend/.env` and set every value. Key settings:

| Variable | Purpose |
| --- | --- |
| `GITHUB_*` | GitHub App client id/secret, app id/slug, private key, webhook secret |
| `ENCRYPTION_KEY` | AES-256 key used to encrypt env vars and OAuth tokens at rest |
| `JWT_SECRET` | HS256 secret that signs session cookies |
| `SITE_BASE_DOMAIN` | The domain deployed sites are served under (wildcard DNS required) |
| `SITE_RESERVED_HOSTS` | Subdomains that are never treated as deployments (`api`, `www`, …) |
| `CORS_ALLOWED_ORIGINS` | Browser origins allowed to call the API |
| `TRUSTED_PROXY_IPS` | Proxy CIDR ranges so rate limits see the real client IP |
| `RATE_*` | Rate limit knobs (global, auth, webhook, build) |
| `BUILD_TIMEOUT` | Per-build timeout in seconds (default 300) |
| `ARTIFACT_RETENTION_DAYS` | How long MinIO artifacts are kept (default 30) |

## Project layout

```
backend/
  cmd/api/        # control-plane API + deployment gateway
  cmd/worker/     # build consumer
  internal/       # api, auth, build, clients, db, envvar, framework,
                  # gateway, github, logging, project, prune, queue,
                  # runner, storage, user
  pkg/            # config, shared types, crypto/jwt/utils
frontend/
  src/
    components/   # layout, custom, and shadcn ui components
    hooks/        # useAuth, useSessionGuard
    pages/        # HomePage, dashboard, project, builds, settings
    shared/       # API client + SSE subscription
    lib/          # formatting and cn() helpers
```

## Docs

- [`docs/architecture.md`](docs/architecture.md) — system design, data flow, data
  model, and security model
- [`docs/deploy.md`](docs/deploy.md) — production deployment, DNS + SSL, GitHub
  App setup, hardening
- [`backend/.env.example`](backend/.env.example) — full configuration template
