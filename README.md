# Status Aggregator

Aggregates third-party status pages (GitHub, depot.dev, Grafana Cloud, Cloudflare, Vercel, and any other Statuspage.io source) into one dashboard. Backend is Go with [Huma](https://huma.rocks); frontend is SvelteKit with Svelte 5 runes.

## Quick start

```bash
# One-time: install frontend deps (requires Node 20+)
cd frontend && npm install --legacy-peer-deps && cd ..

# Run backend (:8080) and frontend (:5173) in two terminals
make dev-backend
make dev-frontend
```

Open http://localhost:5173.

- Backend docs (Swagger UI): http://localhost:8080/docs
- OpenAPI spec: http://localhost:8080/openapi.json

## Configuration

Environment variables read by the backend:

| Var                  | Default             | Description                                                                     |
| -------------------- | ------------------- | ------------------------------------------------------------------------------- |
| `STATUS_ADDR`        | `:8080`             | HTTP listen address                                                             |
| `STATUS_DB_PATH`     | `data/providers.db` | SQLite file                                                                     |
| `STATUS_ADMIN_TOKEN` | _(unset)_           | Bearer token required for mutation endpoints. When unset, mutations return 503. |

Reads (`/api/overview`, `/api/providers/{id}`) are always public.

## Adding a provider

Two ways:

**Via the web UI** — open `/settings`, paste your admin token once, pick a feed kind, enter the base URL, click "Test connection", save.

**Via curl:**

```bash
curl -X POST http://localhost:8080/api/providers \
  -H "Authorization: Bearer $STATUS_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Atlassian",
    "kind": "statuspage_io",
    "params": {"base_url": "https://status.atlassian.com"}
  }'
```

The five most common Statuspage.io pages (GitHub, depot.dev, Grafana Cloud, Cloudflare, Vercel) are seeded on first run.

## Adding a new feed kind

Not every status page uses Statuspage.io — AWS uses RSS, Google Cloud uses a custom JSON endpoint, etc. To support a new format:

1. Create `backend/internal/providers/<yourfeed>.go`.
2. Implement the `providers.Factory` interface (`Kind`, `Label`, `Fields`, `Build`, `Validate`) and a `providers.Provider` (`Config`, `Fetch`).
3. Call `providers.Register(&yourFactory{})` from an `init()`.

The web UI's "Add provider" form automatically discovers your new kind via `GET /api/feed-kinds` and renders the form fields you declared.

The existing `statuspageio.go` is the reference implementation.

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

## Out of scope for v1

Auth beyond the single admin token · notifications · historical uptime graphs · non-Statuspage feed kinds (extend the registry to add RSS / custom JSON).
