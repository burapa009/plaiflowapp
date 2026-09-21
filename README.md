# PlaiFlow Phase 0

Foundation for the responsive dashboard, Go API/worker, PostgreSQL inbound-event queue, and transport-neutral Python export contract.

## Local checks

```powershell
docker compose up -d postgres
docker run --rm --network host -v "${PWD}/api/migrations:/migrations" migrate/migrate:v4.19.1 -path=/migrations -database "postgres://plaiflow:local-only-password@localhost:5432/plaiflow?sslmode=disable" up
cd api; go test ./...; go vet ./...; go build ./cmd/server ./cmd/worker ./cmd/jobworker
cd ../web; pnpm install --frozen-lockfile; pnpm test; pnpm lint; pnpm typecheck; pnpm build
cd ../python; `$env:PYTHONPATH='src'; python -m unittest discover -s tests; python -m plaiflow_export
```

Copy each runtime's `.env.example` to `.env` and fill values locally. Never commit `.env` files. Browser code calls Next.js only; Next.js calls the Go API with the server-only `DASHBOARD_API_TOKEN`.

## Runtime boundaries

- `web/`: Next.js App Router dashboard and application-owned UI primitives.
- `api/cmd/server`: public webhook, operational endpoints, and protected dashboard API.
- `api/cmd/worker`: database-backed processing, heartbeat, retry seam, and retention cleanup.
- `python/`: importable export contract shell; it has no deployed service or business export logic.
- `contracts/export/v1/`: shared opaque request/result fixtures.

The UI implements the repository-owned [PlaiFlow Design System](design-system/plaiflow/MASTER.md) with shared semantic tokens in `web/app/globals.css` and application-owned primitives in `web/components/ui/`.

See [staging runbook](docs/runbooks/staging.md) for deployment, smoke tests, metrics, and rollback.
