# Step 5 — Operate: timesheets + expenses

Documents what Step 5 implements in `api-laravel/`.

## Goal

Time & spend tracking with approval: weekly timesheets, work-from-home
logging, expense categories, manager → accountant expense flow, and
Frankfurter FX conversion. Project task linkage is deferred to Step 6.

## What's implemented

### Time (`Modules/Time`)

- `GET|POST /api/v1/companies/{companyId}/timesheets` — create-or-get Mon–Sun
  week (`date`, optional `employee_id`); self or `timesheets.view`
- `GET .../timesheets/pending` — manager reports, or HR orphans past weeks
  (`timesheets.approve`)
- `GET .../timesheets/{timesheetId}`
- `POST .../timesheets/{timesheetId}/entries` — upsert by day; duration in
  minutes (day ≤ 1440, week ≤ 10080); editable when `open`/`rejected`
- `DELETE .../entries/{entryId}`
- `POST .../submit|approve|reject`
- `PUT .../employees/{employeeId}/work-from-home` — presence/absence row
- `GET|PATCH .../work-from-home` — company `work_from_home_enabled` toggle

### Finance (`Modules/Finance`)

- Expense categories CRUD (`expenses.manage_categories` for writes)
- Default categories seeded on company create
- `POST /expenses` — cents + currency; routes to `manager_approval` or
  `accounting_approval`; Frankfurter FX when currencies differ
- Pending manager / accounting queues; manager/accounting approve|reject
- `POST|DELETE /employees/{employeeId}/accountant` — Spatie `accountant` role
- Unassigning last manager moves orphan expenses to `accounting_approval`

### Dashboard

- `me`: `timesheet_current_week`, `wfh_today` (+ Step 4 widgets)
- `manager`: pending timesheets + expenses
- `hr`: pending orphan timesheets
- `accountant`: `pending_accounting_expenses`

### RBAC

Seeder adds `timesheets.*`, `expenses.*`; role `accountant` with
`expenses.view` + `expenses.finalize`; `manager` gets `timesheets.approve`.

## Verify

```bash
cd api-laravel
composer dump-autoload
php artisan migrate --force
php artisan db:seed --class=RolePermissionSeeder --force
php artisan serve --port=8000
```

## Deferred

Project/task linkage on timesheet entries, PTO, expense receipts.
