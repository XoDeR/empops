# Step 2 — Teams, hierarchy, and org structure

Documents what Step 2 implements in `api-laravel/`.

## Goal

Org chart–aware HRIS core: teams with members/lead, manager/direct-report hierarchy, dashboard shells, and mutation audit logs.

## What's implemented

### Team module (`Modules/Team`)

- CRUD `/api/v1/companies/{companyId}/teams`
- Membership: `POST/DELETE .../teams/{teamId}/members/{employeeId}`
- Lead: `PUT .../teams/{teamId}/lead` (`employee_id` nullable; auto-adds membership)
- Permissions: `teams.view|create|update|delete|manage_members`

### Hierarchy (`Modules/Employee`)

- Tables: `direct_reports`
- `GET .../employees/{id}/managers` + `.../direct-reports`
- `POST .../managers` / `DELETE .../managers/{managerId}` (`hierarchy.assign`)
- Additive Spatie role `manager` synced when report count changes
- Employee payload: `manager`, `managers[]`, `teams[]`, `is_manager`

### Dashboards + audit (`Modules/Company`)

- `GET .../dashboard/me|team|manager|hr` — empty widget shells + flags
- `GET .../audit-logs` (HR/Admin) — Spatie activity log filtered by `company_id`
- Package: `spatie/laravel-activitylog`

### RBAC

Seed via `RolePermissionSeeder`: Step 1 perms + teams/hierarchy; role `manager` (no default perms).

## Verify

```bash
cd api-laravel
composer dump-autoload
php artisan migrate --force
php artisan db:seed --class=RolePermissionSeeder --force
php artisan serve --port=8000

# After login + company: create team, assign member/lead, assign manager
```

## Deferred

Team news, ships, useful links, full dashboard widgets.
