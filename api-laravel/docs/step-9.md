# Step 9 — Hardware & software inventory

Documents what Step 9 implements in `api-laravel/`.

## Goal

Asset register: company hardware (lend/regain) and software licenses
(seats, purchase FX, file attachments).

## What's implemented

### Hardware (`Modules/Hardware`)

- Hardware assets (`name`, `serial_number`, optional `employee_id`)
- List filters: `status=all|available|lent`, search `q`
- Lend / regain endpoints
- Employee assigned hardware (self or `hardware.view`)

- Software licenses with seat pool + `employee_software` pivot
- Purchase amount (minor units) + sync Frankfurter FX into converted fields
- Give / revoke seat; give-all assigns `min(remaining, eligible)`
- Single seat attach returns 422 when no seats remain
- Files via Media Library collection `software` (temporary upload attach)
- Employee assigned software (self or `software.view`)

### RBAC

`hardware.view`, `hardware.manage`, `software.view`, `software.manage` —
seeded for administrator and hr in `RolePermissionSeeder`.

## Verify

```bash
cd api-laravel
composer dump-autoload
php artisan migrate --force
php artisan db:seed --class=RolePermissionSeeder --force
php artisan serve --port=8000
# POST /api/v1/companies/{id}/hardware
# POST /api/v1/companies/{id}/hardware/{id}/lend
# POST /api/v1/companies/{id}/softwares
# POST /api/v1/companies/{id}/softwares/{id}/seats
```

## Deferred

Hardware purchase/cost tracking; license expiry cron; audit log UI.
