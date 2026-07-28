# Step 1 — Auth + RBAC + Companies + Employees

Documents what Step 1 implements in `api-go/`.

## Goal

Go parity with Laravel Step 1: Postgres-backed JWT auth, company/employee modules, RBAC via `employee_roles`.

## What's implemented

### Auth (Core)

- Postgres `users` + `refresh_tokens` (JTI rotation/revocation)
- `POST /api/v1/auth/register|login|refresh|logout`, `GET /api/v1/auth/me`
- bcrypt password hashing (`golang.org/x/crypto/bcrypt`)

### Modules

- `internal/modules/company` — companies, join, invitations
- `internal/modules/employee` — employees, positions, statuses
- `pkg/companyauth` — membership + permission middleware
- Enabled in `config/modules.yaml`: `company`, `employee`

### Migrations

```bash
go run ./cmd/migrate          # applies core, company, employee namespaces
go run ./cmd/migrate status   # list on-disk migrations
```

### SQLC

- Schema: `schema/schema.sql` (DDL only)
- Generated: `coredb`, `companydb`, `employeedb`
- Requires sqlc ≥ 1.29 (`make sqlc`)

### Config

- **Requires** `EMPOPS_DB_DSN` (or `db.dsn` in config) — no in-memory stub in Step 1
- Default Postgres: `postgres://empops:empops@localhost:5433/empops_go?sslmode=disable`

## Verify

```bash
docker compose up -d postgres-go
cd api-go
go run ./cmd/migrate
go run ./cmd/api

curl http://127.0.0.1:8080/api/v1/health
curl -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Bob","email":"bob@empops.local","password":"password","password_confirmation":"password"}'
```

## Deferred

Same auth prod items as Laravel — see `empops-docs/auth-prod-todos.md`.
