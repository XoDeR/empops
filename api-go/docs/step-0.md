# Step 0 — Go API skeleton

This document tracks what the Step 0 scaffold in `api-go/` implements.

## Goal

Stand up a compiling, runnable Go skeleton that follows the modular DDD
layout (flat DDD for Core, module registry for vertical slices) *before* any
real business logic or database wiring lands, so the frontend and other
tooling can integrate against a stable auth/JSON contract immediately.

## What's implemented

### Repo scaffold

- Three binaries: `cmd/api`, `cmd/migrate`, `cmd/worker`.
- Core layers under `internal/`: `domain/{entity,repository}`, `usecase`,
  `adapter/{http,persistence}`, `infrastructure/{config,database}`.
- Shared SDKs under `pkg/`: `module`, `jwt`, `logger`, `response`,
  `pagination`, `uuidv7`, `bus`, `migration` — none of them import
  `internal/*`, per the dependency rules in the architecture doc.
- One vertical module (`internal/modules/example`) demonstrating the
  `IModule` lifecycle (`Initialize` → `Start` → `RegisterRoutes` → `Stop`)
  and the `init()` self-registration pattern used to enable modules via
  `config/modules.yaml` + a blank import in `cmd/api/main.go`.

### HTTP + JSON envelope

Every response uses the shared envelope:

```json
{ "success": true, "message": "...", "data": {}, "error": null, "timestamp": "ISO8601" }
```

Routes (all under `/api/v1`):

- `GET /health` → `{ "status": "ok" }`
- `GET /version` → `{ "version": "0.0.0", "name": "empops-go" }`
- `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `GET /auth/me`
- `GET /example/ping` → `{ "pong": true }` (from the example module)

CORS allows `http://localhost:5173` and `http://localhost:3000`.

### Auth (stub)

`internal/usecase/auth_usecase.go` is an explicit **stub**:

- `Login` accepts any non-empty email/password, lazily creating (or
  reusing) an in-memory `User` for that email — no password hashing or
  verification happens yet.
- Tokens are real `golang-jwt/jwt/v5` HS256 tokens with `sub`, `jti`, `iat`,
  `exp`, `iss` (`empops`), `aud` (`empops-web`) claims, plus a custom
  `type` claim distinguishing access vs refresh tokens.
- Access tokens expire after 15 minutes, refresh tokens after 7 days
  (`config/app.*.yaml` → `jwt.access_token_ttl` / `jwt.refresh_token_ttl`).
- `GET /auth/me` requires a valid Bearer **access** token
  (`internal/adapter/http/middleware/auth.go`); refresh tokens are rejected
  there since they carry `type: refresh`.
- Persistence is `internal/adapter/persistence.MemoryUserRepository`, an
  in-memory `repository.UserRepository` implementation — **no PostgreSQL
  connection is required to run or test auth in Step 0.**

### Database / migrations (stubbed, not required to run)

- `internal/infrastructure/database/postgres.go` provides a `pgxpool`
  connect helper, but nothing in `cmd/api`'s stub flow calls it yet.
- `migrations/core/` has three real migrations (`users`,
  `refresh_tokens`, `roles`/`permissions`/`role_permissions`/`user_roles`)
  matching the domain the Core auth/RBAC use cases will eventually need.
- `pkg/migration` can discover (`Discover`) migration files per namespace;
  `cmd/migrate` uses it to list what's on disk and prints
  `"migrations: use postgres when configured (Step 0 stub)"` — it does not
  connect to or mutate any database yet.
- `sql/core/queries.sql` + `sqlc.yaml` are stubbed out so `sqlc generate`
  has real targets once persistence moves off the in-memory repository.

### Config

- `config/app.dev.yaml` / `config/app.test.yaml`: HTTP port, JWT
  secret/issuer/audience/TTLs, CORS origins, log level, optional DB DSN.
- `config/modules.yaml`: `enabled: [example]`.
- Environment overrides: `EMPOPS_JWT_SECRET`, `EMPOPS_DB_DSN`,
  `EMPOPS_HTTP_PORT`, `EMPOPS_CONFIG`, `EMPOPS_MODULES_CONFIG`,
  `EMPOPS_MIGRATIONS_DIR` (see `.env.example`).

## What's intentionally deferred

- Real password hashing/verification and user registration.
- PostgreSQL-backed `UserRepository` (SQLC), transactions, refresh-token
  revocation storage.
- RBAC middleware/permission checks (tables exist via migration `000003`,
  no use case yet).
- Any module beyond the `example` reference module.
- Redis-backed `pkg/bus` implementation (in-memory only for now).
- Scheduled jobs in `cmd/worker` (heartbeat log only).

## Verifying locally

```bash
cd api-go
go build -o bin/api.exe ./cmd/api
go build -o bin/migrate.exe ./cmd/migrate
go build -o bin/worker.exe ./cmd/worker

go run ./cmd/api
curl http://localhost:8080/api/v1/health
curl -X POST http://localhost:8080/api/v1/auth/login -d '{"email":"a@b.com","password":"x"}'
```
