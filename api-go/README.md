# empops api-go

Go backend for EmpOps, following modular DDD architecture (Clean Architecture
layers for Core + vertical modules for feature slices).

**Step 1** (auth, RBAC, companies, employees), **Step 2** (teams, hierarchy),
and **Step 3** (uploads, media, places) are implemented — see
[`docs/step-1.md`](docs/step-1.md), [`docs/step-2.md`](docs/step-2.md), and
[`docs/step-3.md`](docs/step-3.md). Step 0 skeleton notes remain in
[`docs/step-0.md`](docs/step-0.md).

## Requirements

- Go 1.24+ (module targets Go 1.25 toolchain via `go.mod`; `GOTOOLCHAIN=auto`
  will fetch it automatically if needed)
- PostgreSQL (required for Step 1 — auth, companies, employees)

## Layout

```
cmd/api        composition root: config + JWT + auth + Chi router + module registry
cmd/migrate    namespaced SQL migration runner (core / company / employee)
cmd/worker     background job / cron process (stub: heartbeat only)
config/        app.dev.yaml, app.test.yaml, modules.yaml
migrations/    namespaced *.up.sql / *.down.sql per module (core/, company/, employee/)
schema/        CREATE TABLE DDL for sqlc only (keep in sync with migrations)
sql/           SQLC query sources (core / company / employee)
internal/      Core domain / usecase / adapter / infrastructure + modules/
pkg/           cross-cutting SDKs shared by Core and modules (jwt, logger,
               response, pagination, uuidv7, bus, module, migration)
docs/          architecture and step notes
```

## Running

```bash
go run ./cmd/api
```

The server listens on `:8080` by default (see `config/app.dev.yaml`).
Copy `.env.example` to `.env` to override secrets/ports via environment
variables (`EMPOPS_JWT_SECRET`, `EMPOPS_HTTP_PORT`, `EMPOPS_DB_DSN`, ...).

```bash
go run ./cmd/migrate up   # apply migrations (requires EMPOPS_DB_DSN)
go run ./cmd/worker    # logs a heartbeat every minute until stopped
```

## Building

```bash
make build
# or individually:
go build -o bin/api.exe ./cmd/api
go build -o bin/migrate.exe ./cmd/migrate
go build -o bin/worker.exe ./cmd/worker
```

## SQLC

Requires **sqlc ≥ 1.29** (1.28 OOMs on Windows via wazero). Prefer the
[release binary](https://github.com/sqlc-dev/sqlc/releases) over `go install`
of bleeding-edge tags that need a newer Go toolchain.

```bash
# After editing sql/*/queries.sql or schema/schema.sql:
make sqlc
# or: sqlc generate
```

`sqlc.yaml` reads `schema/schema.sql` (DDL only), not `migrations/` — seeds and
`.down.sql` files must not be in the sqlc schema path.

## API

All routes are under `/api/v1` and return the shared JSON envelope:

```json
{ "success": true, "message": "...", "data": {}, "error": null, "timestamp": "2026-07-27T20:00:00Z" }
```

| Method | Path            | Auth   | Notes                                   |
|--------|-----------------|--------|------------------------------------------|
| GET    | /health         | -      | `{ "status": "ok" }`                     |
| GET    | /version        | -      | `{ "version": "0.0.0", "name": "empops-go" }` |
| POST   | /auth/register  | -      | create account                           |
| POST   | /auth/login     | -      | email + password → JWT + refresh         |
| POST   | /auth/refresh   | -      | rotate refresh token                     |
| POST   | /auth/logout    | -      | revoke refresh token                     |
| GET    | /auth/me        | Bearer | current user                             |
| *      | /companies/...  | Bearer | company CRUD, join, invitations          |
| *      | /companies/{id}/employees/... | Bearer + member | employees, positions, statuses |

Full route list and curl examples: [`docs/step-1.md`](docs/step-1.md).

CORS is open to `http://localhost:5173` and `http://localhost:3000`.

## Modules

Vertical modules live under `internal/modules/<name>` and implement
`pkg/module.IModule`. Enable a module by:

1. Adding it to `config/modules.yaml` under `enabled:`.
2. Blank-importing its package in `cmd/api/main.go` so its `init()` runs.

See `internal/modules/company` and `internal/modules/employee` for Step 1 modules.
