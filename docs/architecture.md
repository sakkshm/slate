# Slate Architecture

Slate is a PaaS that builds static and frontend websites in isolated
containers, stores the output as content-addressed artifacts, and serves it from
an in-process gateway. This document describes the system design: components,
data flow, data model, and security model.

```
                +----------------------------- Control plane (API) ------------------------------+
Git push ──▶ GitHub ──▶ POST /api/webhooks/github ──▶ API ──┬─▶ Postgres (projects, builds, users)
                                                           ├─▶ Redis stream (build jobs)
                                                           ├─▶ Redis Pub/Sub + lists (logs/cancel)
                                                           └─▶ (gateway) host-based routing ──▶ browser
                                                           ▲
                +------------------------------------------┘
                │
        +-------+-----------+       +--------+     +--------+
        │    Worker         ├─▶     │  MinIO │     │ Docker │
        │ (Redis consumer)  │  upload│ (blobs)│     │ (builds│
        +-------------------+       +--------+     +--------+
```

Two long-running Go processes make up the backend:

- **`cmd/api`** — control plane: HTTP API, GitHub OAuth, project/build CRUD,
  encrypted env vars, webhook receiver, SSE log streaming, and the deployment
  gateway (serving deployed sites from MinIO through a local disk cache).
- **`cmd/worker`** — a single build consumer: claims jobs from a Redis stream,
  runs each build in an isolated Docker container, streams logs to Redis,
  uploads the output to MinIO, and publishes a deployment pointer.

Supporting infrastructure: **Postgres** (source of truth), **Redis** (stream,
pub/sub, log buffer, deployment pointer), and **MinIO** (object storage for
build artifacts).

## Process wiring

### API (`backend/cmd/api/main.go`)

On startup the API:

1. Loads config and sets up `slog` logging.
2. Opens Postgres (GORM), Redis, and MinIO via `internal/clients`.
3. Builds a `chi` router with logging, recovery, client-IP resolution
   (`ClientIPFromXFF` when `TRUSTED_PROXY_IPS` is set), and CORS.
4. Wires the **gateway**: a middleware checks `gateway.IsDeploymentHost` and
   short-circuits any request whose host is `<slug>.<SITE_BASE_DOMAIN>`
   straight to the gateway handler — bypassing rate limits and security headers,
   which would break user-controlled content.
5. Applies API-only middleware (security headers, global rate limit) and
   registers routes.
6. Serves HTTP with `ReadHeaderTimeout`/`ReadTimeout`/`IdleTimeout` set and
   **no `WriteTimeout`**, so SSE streams and artifact downloads can stay open.
7. Shuts down gracefully on SIGINT/SIGTERM, stopping the gateway cache pruner
   and draining in-flight requests.

### Worker (`backend/cmd/worker/main.go`)

The worker runs a single blocking consume loop (`ClaimBuildRequest`, consumer
`worker-1`), so the deployment expects exactly one worker. Per job it:

1. Atomically flips the build from `queued` to `building` (fails fast if the
   build was cancelled while queued).
2. Reports `pending` to GitHub as a commit status.
3. Streams build logs to Redis (both a durable list and a pub/sub channel).
4. Runs the build in an isolated Docker container under `BUILD_TIMEOUT`,
   listening on a Redis cancel channel.
5. On success, tars the staging output, computes its **SHA-256**, uploads to
   MinIO (skipping if the hash already exists), marks the build `ready`, and
   publishes a `deploy:<slug>` pointer in Redis.

A background goroutine also runs the **artifact pruner**: every
`PRUNE_INTERVAL_HOURS` it deletes MinIO objects older than
`ARTIFACT_RETENTION_DAYS`, always protecting each project's latest ready build.

## End-to-end deploy flow

1. **Webhook** — a `push` event hits `POST /api/webhooks/github`. The API
   verifies the HMAC-SHA256 signature, matches the repo to a project, checks
   the branch is the prod branch, and dedupes against
   `(project_id, commit_sha)`.
2. **Enqueue** — the API creates a `queued` build row, resolves the framework's
   install/build/out commands, decrypts the project's env vars, and `XADD`s a
   `BuildEvent` JSON payload to the `slate:builds` stream (max length ~1000).
3. **Claim** — the worker `XREADGROUP`s the job, claims it by flipping status to
   `building`, and emits a GitHub commit status.
4. **Build** — the worker runs `runner.RunBuild`: a container from
   `slate-base-runner:latest` (node:20-alpine + git) fetches the exact commit
   (`git fetch --depth 1 <sha> && git checkout FETCH_HEAD`), then runs the
   install and build commands and copies the output to a bind-mounted `/staging`
   directory.
5. **Store** — the staging dir is tarred, hashed (SHA-256), and uploaded to
   MinIO under `projects/<projectID>/builds/<hash>.tar.gz`. The build is marked
   `ready` and `deploy:<slug>` is set to `{project_id, asset_hash, updated_at}`
   with a 7-day TTL.
6. **Serve** — a request to `https://<slug>.<SITE_BASE_DOMAIN>` is matched by
   the gateway. It resolves the current asset (Redis first, Postgres fallback),
   downloads and extracts the artifact into a local disk cache, and serves the
   files with SPA `index.html` fallback and `http.ServeContent` range support.

### Redis protocol

| Key/channel | Kind | Purpose |
| --- | --- | --- |
| `slate:builds` | Stream (group `workers`) | Build job queue (`XADD` / `XREADGROUP` / `XACK`) |
| `deploy:<slug>` | String (7-day TTL) | Current `asset_hash` for a project, written by worker, read by gateway |
| `slate:logs:<buildID>` | List (24h TTL) | Durable log buffer for late-joining SSE clients |
| `slate:logs:<buildID>` | Pub/Sub channel | Live log-line fan-out |
| `slate:build-done:<buildID>` | Pub/Sub channel | Signals SSE streams to end |
| `slate:cancel:<buildID>` | Pub/Sub channel | API → worker cancel requests |

## The gateway (`backend/internal/gateway`)

The gateway serves every deployed site from the API process on the same port,
using **host-based routing**:

- **Host matching** — `IsDeploymentHost` requires the host to end with
  `.<SITE_BASE_DOMAIN>`, the leading label to be a valid slug
  (lowercase alphanumerics/hyphens), and not to be in `SITE_RESERVED_HOSTS`
  (`api`, `www`, `app`, `dashboard`, …). Matched hosts bypass all API
  middleware.
- **Deployment resolution** — `Resolver.GetDeployment` reads `deploy:<slug>`
  from Redis; if missing it falls back to the project's latest `ready` build in
  Postgres and back-fills Redis.
- **Artifact cache** — `ensureArtifact` downloads `projects/<pid>/builds/<hash>.tar.gz`
  from MinIO and extracts it to `/tmp/slate-deploy/<pid>/<hash>` with an atomic
  `.slate-ready` marker (extract to a temp dir, then rename). Content addressing
  means redeploys of identical output share cache entries.
- **Path safety** — archive extraction rejects entries escaping the destination
  and skips symlinks/hardlinks; `serveDeployment` applies the same check to
  request paths.
- **Pruning** — a background pruner evicts cache entries older than 24h every
  30 minutes.

## Data model

GORM-managed tables (`internal/db`):

- **`users`** — keyed by GitHub user ID; username, installation ID, profile,
  and the **encrypted** OAuth access token (AES-256-GCM).
- **`projects`** — UUID PK, owner, unique-per-owner slug, GitHub repo reference,
  prod branch, framework, root dir, and install/build/out commands.
- **`builds`** — UUID PK, project, commit SHA/message, status
  (`queued`/`building`/`ready`/`failed`/`cancelled`), duration, log content, and
  the content-addressed `asset_location` (SHA-256 hash).
- **`project_env_vars`** — `(project_id, key)` unique pair; values stored as
  ciphertext and never returned by the API (masked in responses).

`internal/db.New` runs `AutoMigrate` plus small data migrations (dropping a
retired column, sanitizing legacy slugs).

## Authentication & secrets

- **GitHub OAuth** — `GET /api/auth/github/initiate-login` sets an
  `HttpOnly` state cookie and returns the GitHub authorize URL. The callback
  validates the state, exchanges the code, fetches the profile, requires an app
  installation, encrypts the access token, and issues a 7-day HS256 **JWT**
  session cookie (`slate-session`).
- **Installation tokens** — the API signs a short-lived GitHub App JWT
  (RS256) and exchanges it for per-installation access tokens used for all repo,
  branch, content, and commit-status calls.
- **Encryption** — AES-256-GCM with a random nonce per encryption (prepended).
  Used for both user access tokens and project env vars. `ENCRYPTION_KEY` must
  be a unique random value.
- **Frontend** — cookie-based sessions (`credentials: include`); no tokens in
  `localStorage`. The dashboard guards routes client-side and a global
  `slate:unauthorized` window event bounces expired sessions home.

## Build isolation (`backend/internal/runner`)

Each build runs in a container from `slate-base-runner:latest` (`node:20-alpine`
+ git) with:

- **Network isolation** — a dedicated bridge network
  (`ci_isolated_sandbox`) with inter-container communication disabled.
- **Resource limits** — 4 GiB memory, 2 CPUs, 100 process limit, OOM killer
  active.
- **Filesystem hardening** — read-only rootfs; only the `/staging` bind mount
  and `noexec`/`nosuid` tmpfs mounts (`/tmp`, `/root`, `/app`) are writable.
- **Privilege hardening** — `no-new-privileges`, every Linux capability dropped,
  and a `tini` init to reap zombies.
- **Commit pinning** — the build checks out exactly the pushed commit SHA
  (`git fetch --depth 1`), so builds are deterministic per commit.
- **Cleanup** — the container is force-removed (volumes included) and staging is
  deleted on every exit path.

## Frontend (`frontend`)

A Vite + React 19 + TypeScript SPA using Tailwind v4 (shadcn/Base UI) with
`react-router-dom` and `next-themes`.

- **Auth** — `GithubButton` → OAuth redirect → `GitHubCallback` POSTs the code,
  the server sets the session cookie, and the dashboard loads. `useAuth` +
  `DashboardLayout` redirect unauthenticated users home.
- **API client** (`src/shared/api.ts`) — a thin `fetch` wrapper
  (`credentials: include`, normalized error shapes) plus `subscribeSSE` for
  EventSource consumption.
- **Live logs** — `BuildDetailPage` polls build status every 3s while the build
  is non-terminal and streams logs through `LogViewer` over SSE; the server's
  `done` event stops streaming and refreshes state.
- **State** — no state library; hand-rolled `useState`/`useEffect` per page.
  Route-param changes trigger page remounts via `key` props in `App.tsx`.
- **Styling** — semantic OKLCH CSS variables for light/dark themes (class-based,
  FOUC-prevention script in `index.html`), Space Grotesk variable font.

## Security model

- **Webhooks** — HMAC-SHA256 signature verification, 5 MB body cap, only `push`
  events, dedup by commit. Ignored (but acked) events still return 200 so
  GitHub doesn't retry.
- **Rate limits** — in-memory, per client IP: global 50 rps, auth 10 rpm,
  webhook 100 rpm; build triggers are limited per **user** (10/hour).
  `TRUSTED_PROXY_IPS` is required behind a reverse proxy so limits key off the
  real client IP.
- **HTTP** — 1–5 MB body caps, read/idle timeouts (no write timeout, required
  for SSE/downloads), and strict security headers (CSP, `X-Frame-Options`,
  `nosniff`, HSTS when HTTPS) — applied to API responses only, never to
  deployment traffic.
- **Secrets at rest** — AES-256-GCM for user tokens and env vars; env var values
  are masked in all API responses.
- **Compartmentalization** — the API runs as a non-root user and never mounts
  the Docker socket; only the worker can drive the Docker engine, and its
  builds are sandboxed (read-only rootfs, dropped caps, isolated network,
  memory/CPU/pid limits).

## Reliability & ops

- **Health** — `GET /health` reports Postgres, Redis, and MinIO reachability
  (200/503), used by the prod compose healthcheck.
- **Graceful shutdown** — both processes cancel in-flight work on SIGTERM; the
  API drains with a 5s deadline, the worker aborts running Docker builds.
- **Logging** — `slog`, JSON in production, human-readable in development, each
  component tagged with `component=api` / `component=worker`.
- **Pruners** — two independent jobs: the worker prunes MinIO artifacts
  (30-day default retention) and the API prunes the gateway disk cache (24h
  TTL), keeping the host clean without removing live deployments.
- **CI** — GitHub Actions builds, vets, and tests the backend (`go test -race`
  with Postgres/Redis/MinIO service containers) and typechecks, lints, and
  builds the frontend.
