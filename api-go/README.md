# empops api-go

Go backend for EmpOps, following modular DDD architecture (Clean Architecture
layers for Core + vertical modules for feature slices). This is the **Step 0**
skeleton — see [`docs/step-0.md`](docs/step-0.md) for exactly what it implements.

## Requirements

- Go 1.24+ (module targets Go 1.25 toolchain via `go.mod`; `GOTOOLCHAIN=auto`
  will fetch it automatically if needed)
- PostgreSQL (optional for Step 0 — the stub auth flow works without it)

## Layout

```
cmd/api        composition root: config + JWT + stub auth + Chi router + module registry
cmd/migrate    namespaced SQL migration runner (stub: lists migrations, no DB yet)
cmd/worker     background job / cron process (stub: heartbeat only)
config/        app.dev.yaml, app.test.yaml, modules.yaml
migrations/    namespaced *.up.sql / *.down.sql per module (starts with core/)
sql/           SQLC query sources (sqlc.yaml points at sql/core/queries.sql)
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
go run ./cmd/migrate   # lists discovered migrations (Step 0 stub, no DB writes)
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

## API

All routes are under `/api/v1` and return the shared JSON envelope:

```json
{ "success": true, "message": "...", "data": {}, "error": null, "timestamp": "2026-07-27T20:00:00Z" }
```

| Method | Path            | Auth   | Notes                                   |
|--------|-----------------|--------|------------------------------------------|
| GET    | /health         | -      | `{ "status": "ok" }`                     |
| GET    | /version        | -      | `{ "version": "0.0.0", "name": "empops-go" }` |
| POST   | /auth/login     | -      | stub: accepts any email/password         |
| POST   | /auth/refresh   | -      | issues a new access token                |
| POST   | /auth/logout    | -      | no-op success (no revocation store yet)  |
| GET    | /auth/me        | Bearer | returns the stub user for the token      |
| GET    | /example/ping   | -      | `{ "pong": true }` from the example module |

CORS is open to `http://localhost:5173` and `http://localhost:3000`.

## Modules

Vertical modules live under `internal/modules/<name>` and implement
`pkg/module.IModule`. Enable a module by:

1. Adding it to `config/modules.yaml` under `enabled:`.
2. Blank-importing its package in `cmd/api/main.go` so its `init()` runs.

See `internal/modules/example` for the minimal reference implementation.
