# Step 8 — Grow (engagement & performance)

Documents what Step 8 implements in `api-laravel/`.

## Goal

Manager–employee development toolkit: daily morale with company/team
history jobs, one-on-ones, rate-your-manager surveys, skills directory,
e-coffee pairing, and discipline cases with file attachments.

## What's implemented

### Grow (`Modules/Grow`)

- Daily morale (`emotion` 1–3, one per employee per day) + company/team
  history snapshots
- One-on-ones (talking points, action items, notes); lazy open entries;
  mark happened creates the next entry; unchecked action items carry
  over as talking points
- Rate-your-manager surveys + answers (`bad` / `average` / `good`)
- Skills directory; attach find-or-creates company skill; detach removes
  orphan skills
- e-Coffee: company `e_coffee_enabled` flag; pairing sessions; mark match
  happened
- Discipline cases + events; files via Media Library collection
  `discipline` (temporary upload attach)

### Jobs (`routes/console.php`)

| Command | Schedule |
|---|---|
| `empops:log-company-morale` | daily 23:00 |
| `empops:log-team-morale` | daily 23:00 |
| `empops:rate-manager-start` | daily 01:00 when last day of month |
| `empops:rate-manager-stop` | hourly |
| `empops:e-coffee-start` | weekly Mon 09:00 (enabled companies only) |

### Dashboard widgets

- Me: `morale_today`, `one_on_one_current`, `rate_manager_pending`,
  `e_coffee_current`
- Manager: `one_on_ones_open`, `discipline_active`
- HR: `discipline_active`

### RBAC

`morale.*`, `one_on_ones.*`, `rate_manager.*`, `skills.*`, `e_coffee.*`,
`discipline.*` — seeded in `RolePermissionSeeder` for admin/HR/manager/
employee.

## Verify

```bash
cd api-laravel
composer dump-autoload
php artisan migrate --force
php artisan db:seed --class=RolePermissionSeeder --force
php artisan serve --port=8000
# POST /api/v1/companies/{id}/morale
# php artisan empops:log-company-morale
# php artisan empops:rate-manager-start
# php artisan empops:e-coffee-start
```

## Deferred

OKRs; richer morale analytics; email digests for rate-your-manager.
