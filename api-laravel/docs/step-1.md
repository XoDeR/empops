# Step 1 — Auth + RBAC + Companies + Employees

Documents what Step 1 implements in `api-laravel/`.

## Goal

Multi-tenant employee directory with real login, company tenancy, Spatie RBAC on `Employee`, and Adminland APIs.

## What's implemented

### Auth (`Modules/Auth`)

- `POST /api/v1/auth/register` — name/email/password (confirmed)
- `POST /api/v1/auth/login` — bcrypt verify
- `POST /api/v1/auth/refresh` — rotate refresh token (revoke old `jti`)
- `POST /api/v1/auth/logout` — revoke refresh token
- `GET /api/v1/auth/me` — JWT + loaded User
- `refresh_tokens` table stores active refresh JTIs

### RBAC

- Spatie roles on **`Employee`**: `administrator`, `hr`, `employee`
- Seeded permissions via `database/seeders/RolePermissionSeeder.php`
- Middleware `EnsurePermission` checks employee permissions

### Company (`Modules/Company`)

- `GET/POST /api/v1/companies`, `POST /api/v1/companies/join`
- `GET/PATCH /api/v1/companies/{companyId}`
- `POST /api/v1/invitations/{link}/accept`
- Default currency **EUR**

### Employee (`Modules/Employee`)

- CRUD `/api/v1/companies/{companyId}/employees` + invite
- CRUD `/positions`, `/employee-statuses`
- Self-update allowed for name; HR/Admin for full fields

### Deferred (see `empops-docs/auth-prod-todos.md`)

- Password reset, email verification, 2FA, invite emails, rate limits

## Verify

```bash
cd api-laravel
php artisan migrate --force
php artisan db:seed --class=RolePermissionSeeder --force
php artisan serve --port=8000

# register → login → create company → list employees
curl -X POST http://127.0.0.1:8000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@empops.local","password":"password","password_confirmation":"password"}'
```
