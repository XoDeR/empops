# Step 0 — Laravel API skeleton

Documents what Step 0 implements in `api-laravel/`.

## Goal

“Hello EmpOps” platform skeleton: bootable Laravel API with nwidart modules, JWT auth stubs, Spatie Permission + Media Library wired, shared JSON envelope, CORS for the React SPA.

## What's implemented

### Platform

- Laravel 13 app scaffold in `api-laravel/`
- Packages: `nwidart/laravel-modules`, `spatie/laravel-permission`, `spatie/laravel-medialibrary`, `firebase/php-jwt`
- Modules: `Modules/Auth`, `Modules/Core` (enabled via `modules_statuses.json`)
- `User` model uses `HasUuids` (UUID v7), `HasRoles`, and `InteractsWithMedia`
- Custom `Role` / `Permission` models extend Spatie with `HasUuids`
- Migrations use `uuid('id')->primary()` and UUID morph/FK columns for Spatie pivots
- `AppServiceProvider` calls `Builder::morphUsingUuids()`
- Config: `config/jwt.php`, `config/cors.php`, published `permission` + `media-library` configs
- Env: PostgreSQL default (`empops_laravel` on `:5432`); Go uses a separate instance (`empops_go` on `:5433`)

### HTTP contract

JSON envelope (same shape as Go):

```json
{ "success": true, "message": "...", "data": {}, "error": null, "timestamp": "ISO8601" }
```

Routes (module API prefix `api` + route `v1/...`):

| Method | Path | Notes |
|---|---|---|
| GET | `/api/v1/health` | Core module |
| GET | `/api/v1/version` | `{ name: empops-laravel, version: 0.0.0 }` |
| POST | `/api/v1/auth/login` | Stub — any email/password → JWT pair + user |
| POST | `/api/v1/auth/refresh` | Stub — refresh token → new pair |
| POST | `/api/v1/auth/logout` | Stub success |
| GET | `/api/v1/auth/me` | Requires `Authorization: Bearer <access>` |

JWT claims: `sub`, `jti`, `iat`, `exp`, `iss` (`empops`), `aud` (`empops-web`), `type` (`access`|`refresh`). HS256 via `firebase/php-jwt`.

### Auth module layout

- `Modules/Auth/app/Services/JwtService.php`
- `Modules/Auth/app/Http/Middleware/AuthenticateJwt.php`
- `Modules/Auth/app/Http/Controllers/AuthController.php`
- `Modules/Auth/routes/api.php`

### Intentionally deferred

- Real users, password hashing, email verification, 2FA
- OAuth / social logins
- RBAC enforcement beyond package install
- Media uploads (Media Library package installed; media migration not published yet)
- Full OpenAPI codegen (stub lives in `packages/api-types`)

## Verify

```bash
cd api-laravel
php artisan serve --port=8000
curl http://127.0.0.1:8000/api/v1/health
curl -X POST http://127.0.0.1:8000/api/v1/auth/login -H "Content-Type: application/json" -d "{\"email\":\"dev@empops.local\",\"password\":\"secret\"}"
```
