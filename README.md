# EmpOps

API-first HR operations platform — Laravel and Go backends with a React SPA.

| Path | Role |
|---|---|
| `api-laravel/` | Laravel + nwidart modules backend |
| `api-go/` | Go DDD / modular parity backend (Chi + SQLC) |
| `web-react/` | React 19 + Vite SPA |
| `packages/api-types/` | Shared OpenAPI skeleton |
| `docker-compose.yml` | Local Postgres (Laravel + Go, separate volumes) |

## Step 0

Platform skeleton — see each part’s `docs/step-0.md`:

- [api-laravel/docs/step-0.md](api-laravel/docs/step-0.md)
- [api-go/docs/step-0.md](api-go/docs/step-0.md)
- [web-react/docs/step-0.md](web-react/docs/step-0.md)

## Step 1 — Auth + RBAC + Companies + Employees

Multi-tenant employee directory with login, company tenancy, RBAC, and Adminland.

- [api-laravel/docs/step-1.md](api-laravel/docs/step-1.md)
- [api-go/docs/step-1.md](api-go/docs/step-1.md)
- [web-react/docs/step-1.md](web-react/docs/step-1.md)
- OpenAPI: [packages/api-types/openapi/v1.yaml](packages/api-types/openapi/v1.yaml)
- Auth prod TODOs: [empops-docs/auth-prod-todos.md](../empops-docs/auth-prod-todos.md)

## Step 2 — Teams, hierarchy, and org structure

Org chart–aware HRIS: teams (members/lead), manager/direct reports, dashboard shells, audit logs.

- [api-laravel/docs/step-2.md](api-laravel/docs/step-2.md)
- [api-go/docs/step-2.md](api-go/docs/step-2.md)
- [web-react/docs/step-2.md](web-react/docs/step-2.md)
- OpenAPI: [packages/api-types/openapi/v1.yaml](packages/api-types/openapi/v1.yaml)

### Quick start

```bash
# Postgres — Laravel :5432, Go :5433 (separate volumes)
docker compose up -d postgres-laravel postgres-go

# Laravel API :8000
cd api-laravel && composer install && cp .env.example .env && php artisan key:generate
php artisan migrate --force && php artisan db:seed --class=RolePermissionSeeder --force
php artisan serve

# Go API :8080 (parity)
cd api-go && go run ./cmd/migrate && go run ./cmd/api

# React :5173
cd web-react && npm install && cp .env.example .env && npm run dev
```

| Service | Host port | Database | Volume |
|---|---|---|---|
| `postgres-laravel` | 5432 | `empops_laravel` | `empops_laravel_pg` |
| `postgres-go` | 5433 | `empops_go` (+ `empops_go_test`) | `empops_go_pg` |
