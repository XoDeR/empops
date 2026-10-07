# EmpOps

API-first HR operations platform — Laravel and Go backends with a React SPA.

| Path | Role |
|---|---|
| `api-laravel/` | Laravel + nwidart modules backend |
| `api-go/` | Go DDD / modular parity backend (Chi + SQLC) |
| `web-react/` | React 19 + Vite SPA |
| `packages/api-types/` | Shared OpenAPI skeleton |
| `docker-compose.yml` | Local Postgres (Laravel + Go, separate volumes) |

## Features

Features cover company and employee directories, teams and org hierarchy, media uploads, places, notifications, time off, worklogs, flows, finance, recruiting, growth, hardware and software inventory, billing, and a wiki. A shared OpenAPI spec lives in packages/api-types, and Docker Compose provides local Postgres.

## Laravel API libraries
- nwidart/laravel-modules         Modular structure (Modules/Auth, Team, etc.)
- firebase/php-jwt                JWT auth (the custom AuthenticateJwt middleware)
- spatie/laravel-permission       Roles and permissions (RBAC)
- spatie/laravel-medialibrary     File and media uploads
- spatie/laravel-activitylog      Audit log

### Quick start

```bash

# Laravel API :8000
# Postgres — Laravel :5432 (separate volumes)
docker compose up -d postgres-laravel
cd api-laravel && composer install && cp .env.example .env && php artisan key:generate
php artisan migrate --force && php artisan db:seed --class=RolePermissionSeeder --force
php artisan serve

# Go API :8080 (parity)
# Postgres — Go :5433 (separate volumes)
docker compose up -d postgres-go
cd api-go && go run ./cmd/migrate && go run ./cmd/api

# React :5173
cd web-react && npm install && cp .env.example .env && npm run dev
```

| Service | Host port | Database | Volume |
|---|---|---|---|
| `postgres-laravel` | 5432 | `empops_laravel` | `empops_laravel_pg` |
| `postgres-go` | 5433 | `empops_go` (+ `empops_go_test`) | `empops_go_pg` |
