# Status Aggregator

Aggregates third-party status pages into one dashboard. Backend is Go with [Huma](https://huma.rocks); frontend is SvelteKit with Svelte 5 runes.

## Supported feed kinds

| Kind            | Label           | Use for                                                                        |
| --------------- | --------------- | ------------------------------------------------------------------------------ |
| `statuspage_io` | Statuspage.io   | GitHub, Cloudflare, Vercel, Atlassian, and any other Statuspage.io-hosted page |
| `rss`           | RSS / Atom feed | Pages that publish incidents as a feed — Slack, Google Workspace, etc.         |
| `auth0`         | Auth0           | https://status.auth0.com (scrapes the embedded Next.js payload)                |

## Quick start

```bash
# One-time: install frontend deps (requires Node 20+)
cd frontend && npm install && cd ..

# Run backend (:8080) and frontend (:5173) in two terminals
make dev-backend
make dev-frontend
```

Open http://localhost:5173.

- Backend docs (Swagger UI): http://localhost:8080/docs
- OpenAPI spec: http://localhost:8080/openapi.json

## Configuration

Environment variables read by the backend:

| Var                             | Default             | Description                                                                                       |
| ------------------------------- | ------------------- | ------------------------------------------------------------------------------------------------- |
| `STATUS_ADDR`                   | `:8080`             | HTTP listen address                                                                               |
| `STATUS_DB_PATH`                | `data/providers.db` | SQLite file                                                                                       |
| `STATUS_ADMIN_TOKEN`            | _(unset)_           | Bearer token required for mutation endpoints. When unset, mutations return 503.                   |
| `STATUS_CORS_ORIGIN`            | _(unset)_           | Comma-separated origin allowlist. Unset disables CORS (prod same-origin). Set in dev for `:5173`. |
| `STATUS_ALLOW_PRIVATE_NETWORKS` | `false`             | Allow outbound fetches to loopback/RFC1918/link-local. Enable only for internal status pages.     |

Reads (`/api/overview`, `/api/providers/{id}`) are always public.

## Adding a provider

Two ways:

**Via the web UI** — open `/settings`, paste your admin token once, pick a feed kind, enter the base URL, click "Test connection", save.

**Via curl:**

```bash
# Statuspage.io
curl -X POST http://localhost:8080/api/providers \
  -H "Authorization: Bearer $STATUS_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Atlassian",
    "kind": "statuspage_io",
    "params": {"base_url": "https://status.atlassian.com"}
  }'

# RSS / Atom
curl -X POST http://localhost:8080/api/providers \
  -H "Authorization: Bearer $STATUS_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Slack",
    "kind": "rss",
    "params": {"feed_url": "https://slack-status.com/feed/rss", "active_hours": "24"}
  }'

# Auth0 (no params)
curl -X POST http://localhost:8080/api/providers \
  -H "Authorization: Bearer $STATUS_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Auth0", "kind": "auth0", "params": {}}'
```

The five most common Statuspage.io pages (GitHub, depot.dev, Grafana Cloud, Cloudflare, Vercel) are seeded on first run.

## Adding a new feed kind

Not every status page uses one of the built-in shapes. To add another:

1. Create `backend/internal/providers/<yourfeed>.go`.
2. Implement the `providers.Factory` interface (`Kind`, `Label`, `Fields`, `Build`) and a `providers.Provider` (`Config`, `Fetch`).
3. Call `providers.Register(&yourFactory{})` from an `init()`.

The web UI's "Add provider" form automatically discovers your new kind via `GET /api/feed-kinds` and renders the form fields you declared.

Reference implementations: `statuspageio.go` (single URL param, JSON API), `rss.go` (multi-param, XML feed with classification heuristics), `auth0.go` (zero-param scraper for a fixed endpoint).

## Regenerating frontend types

When you change backend API shapes, regenerate the TS types:

```bash
# With backend running:
make gen
```

This runs `openapi-typescript` against `/openapi.json`.

## Polling

- Backend polls every 60s with a 10s per-provider timeout, in parallel.
- Frontend refreshes `/api/overview` every 30s.
- On fetch failure, the cache keeps the last-known status and marks it `stale` after `3 * poll_interval`.

## Docker

A multi-stage `Dockerfile` at the repo root builds the frontend, embeds it in the Go binary, and ships a distroless image. CI publishes it to `ghcr.io/bueti/status-aggregator` on every push to `main` and on version tags (`v*`).

```bash
docker run --rm -p 8080:8080 \
  -e STATUS_ADMIN_TOKEN=change-me \
  -v status-data:/data \
  ghcr.io/bueti/status-aggregator:latest
```

Local build:

```bash
make docker-build
make docker-run          # runs the image on :8080 with a local ./data volume
```

## Helm

A chart lives at `charts/status-aggregator`. It ships a single-replica `Deployment` (SQLite is a single-writer store), a `PersistentVolumeClaim` for `/data`, a `Service`, an optional `Ingress`, and a `Secret` for the admin token.

On tag push, CI publishes the chart as an OCI artifact to `ghcr.io/bueti/charts/status-aggregator`:

```bash
helm install status \
  oci://ghcr.io/bueti/charts/status-aggregator \
  --version 0.1.0 \
  --set adminToken.value=change-me
```

Or install from the local repo checkout:

```bash
helm install status charts/status-aggregator \
  --set adminToken.value=change-me \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=status.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix
```

See `charts/status-aggregator/values.yaml` for the full set of knobs.
