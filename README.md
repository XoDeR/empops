# EmpOps

API-first HR operations platform — Laravel and Go backends with a React SPA.

| Path | Role |
|---|---|
| `api-laravel/` | Laravel + nwidart modules backend |
| `api-go/` | Go DDD / modular parity backend (Chi + SQLC) |
| `web-react/` | React 19 + Vite SPA |
| `packages/api-types/` | Shared OpenAPI skeleton |
| `docker-compose.yml` | Local Postgres |

## Step 0

Platform skeleton — see each part’s `docs/step-0.md`:

- [api-laravel/docs/step-0.md](api-laravel/docs/step-0.md)
- [api-go/docs/step-0.md](api-go/docs/step-0.md)
- [web-react/docs/step-0.md](web-react/docs/step-0.md)

### Quick start

```bash
# Laravel API :8000
cd api-laravel && composer install && cp .env.example .env && php artisan key:generate && php artisan serve

# Go API :8080 (optional parity)
cd api-go && go run ./cmd/api

# React :5173
cd web-react && npm install && cp .env.example .env && npm run dev
```
