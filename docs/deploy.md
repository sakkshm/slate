# Slate Production Deployment

Slate serves every deployed site from the API itself (in-process gateway, host-based
routing) on a single port. A project with slug `my-site` is served at
`http(s)://my-site.<SITE_BASE_DOMAIN>`.

## Production environment (`APP_ENV_PROD=production`)

Everything runs through the `prod` compose profile:

```bash
cd backend
# provision required env vars (see .env.example values in backend/.env)
docker compose --profile prod up -d --build
```

Services brought up by `--profile prod`:

| Service     | Notes                                                                 |
|-------------|-----------------------------------------------------------------------|
| `api-prod`  | Alpine prod image; **runs as non-root `slate` user, no Docker socket**; healthcheck `wget /health`; `restart: unless-stopped` |
| `worker-prod`| Alpine prod image; process healthcheck (`kill -0 1`); restart policy. **Runs as root and mounts the Docker socket** — this is required to launch build containers; the socket is never mounted into the API |
| `db`, `redis`, `minio` | persisted volumes; `restart: unless-stopped` + healthchecks |

Prod images are multi-stage (`target: production`) — distroless-style `alpine:3.20`
runtime with CA certs (`ca-certificates`) so the worker can reach the GitHub API and
`wget` is available for healthchecks. Build context requires `scripts/` and `go.mod`.

Artifacts are retained `ARTIFACT_RETENTION_DAYS` (default 30) and pruned by the worker
every `PRUNE_INTERVAL_HOURS` (default 1).

## DNS + SSL

The gateway matches any host `<slug>.<SITE_BASE_DOMAIN>`, so you need a **wildcard DNS
record** pointing at the server:

```
*.slate.dev   A   <server-ip>
```

Serve `SITE_SCHEME=http` for plain HTTP, or terminate TLS in front (Caddy, Traefik,
nginx, or a load balancer) and set:

```env
SITE_SCHEME=https
SITE_BASE_DOMAIN=slate.dev
SITE_PORT=443            # or omit when the frontend derives it
SITE_RESERVED_HOSTS=api,www,app,dashboard,admin,minio,docs,git,status
```

`SITE_RESERVED_HOSTS` is a comma-separated list of host labels that are never treated
as deployments (API, control plane, etc.). A request for `api.slate.dev` routes to the
API; anything else matching `<slug>.slate.dev` serves that project's latest build.

## GitHub integration

The API must be reachable by GitHub to receive push webhooks:
`APP_URL` (used to build the webhook URL `{APP_URL}/api/webhooks/github`) must be a
public HTTPS URL. Configure the GitHub App's callback + webhook URLs accordingly.

## CI

`.github/workflows/ci.yml` runs on push/PR to `main`:

- **Backend**: `go build`, `go vet`, `go test -race ./...` with Postgres/Redis/MinIO
  service containers. Integration-gated tests (`SLATE_TEST_MINIO=1`, `SLATE_TEST_DB=1`)
  run against those services.
- **Frontend**: pnpm install (frozen lockfile), `tsc -b`, `eslint`, `vite build`.

## Security hardening

The API applies the following protections to **API traffic only** — the deployment
gateway short-circuits before these middleware, so deployed sites are never throttled
or stamped with API security headers.

### Rate limiting

Requests are keyed by client IP (`RATE_GLOBAL_RPS`, `RATE_AUTH_RPM`, `RATE_WEBHOOK_RPM`)
or per user (`RATE_BUILD_RPH`), using a fixed-window limiter. When a limit is exceeded
the API returns `429 Too Many Requests`.

| Var | Default | Applies to |
|-----|---------|-----------|
| `RATE_GLOBAL_RPS` | `50` | all API requests, per second per IP |
| `RATE_AUTH_RPM` | `10` | OAuth initiate/callback, per minute per IP |
| `RATE_WEBHOOK_RPM` | `100` | GitHub webhook receiver, per minute per IP |
| `RATE_BUILD_RPH` | `10` | build triggers, per hour per user |

> **`TRUSTED_PROXY_IPS` is required if a reverse proxy (Caddy, Traefik, nginx) or load
> balancer terminates TLS in front of the API.** Set it to the proxy's CIDR ranges
> (e.g. `TRUSTED_PROXY_IPS=10.0.0.0/8,127.0.0.1/32`) so the API resolves the real
> client IP from `X-Forwarded-For`. Without it every client appears to come from the
> proxy's IP and the per-IP limits collapse into a single shared bucket — most
> noticeably `RATE_AUTH_RPM=10` would limit the *whole site* to 10 logins/min.
> When unset, Slate trusts only the TCP peer (`RemoteAddr`), which is correct for
> direct or same-host-proxy setups.

### CORS

`CORS_ALLOWED_ORIGINS` (comma-separated) controls which browser origins may call the
API. Include the frontend origin exactly (`scheme://host:port`). Responses to other
origins carry no CORS headers. Only the configured origins are allowed; credentials
(cookies) are enabled.

### Requests

- **Body caps** via `http.MaxBytesReader`: 5 MB on the GitHub webhook, 1 MB on the
  OAuth callback.
- **Timeouts**: `ReadHeaderTimeout` 5s, `ReadTimeout` 10s, `IdleTimeout` 60s. There is
  deliberately **no `WriteTimeout`**, because SSE log streams and artifact downloads
  must be able to stay open for their full duration.
- **Security headers** on every API response: strict `Content-Security-Policy`,
  `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy`.

### Container boundaries

- `api-prod` runs as the non-root `slate` user and mounts **no** Docker socket — the
  API only publishes to Redis; it never talks to the Docker engine.
- `worker-prod` is the only service that mounts `/var/run/docker.sock`; it must run as
  root to drive the Docker engine API and launch isolated build containers. Keep the
  worker host locked down (single-purpose host, minimal other services).

## Operations

- **Health**: `GET /health` returns `200 {"status":"ok",...}` when DB, Redis and MinIO
  are reachable, `503` otherwise. Used by the api-prod healthcheck.
- **Logging**: `slog` — JSON in production, human-readable text in development. Each
  component tags lines with `component=api` / `component=worker`.
- **Deployment cache**: extracted sites live under `/tmp/slate-deploy/<slug>/<hash>`
  with a 24h TTL; the gateway falls back to Postgres + MinIO when a cache entry is
  missing. `DELETE /api/projects/{id}` removes the deploy key, MinIO artifacts and DB
  rows; cache entries are removed by the background pruner.
