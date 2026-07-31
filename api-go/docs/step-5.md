# Step 5 — Operate: timesheets + expenses

Documents what Step 5 implements in `api-go/`.

## Goal

Go parity with the Step 5 Operate contract: weekly timesheets, WFH,
expense categories, manager → accountant expenses, and Frankfurter FX.

All new routes live under `/api/v1/companies/{companyId}`, use the shared
`pkg/response` envelope, and are guarded by `httpauth.RequireAuth` +
`companyauth.RequireMember` (+ `RequirePermission` where noted).

## What's implemented

### Time (`internal/modules/time`)

- `GET|POST /timesheets` — create-or-get Mon–Sun week
- `GET /timesheets/pending` — manager reports; HR/admin past weeks without
  managers
- Entries upsert by day; caps 1440 / 10080 minutes
- Submit / approve / reject status machine
- WFH put + company toggle (`companies.work_from_home_enabled`)

Schema: `migrations/time/000001_create_timesheets.up.sql`,
`migrations/company/000004_add_work_from_home_enabled.up.sql`.

### Finance (`internal/modules/finance`)

- Categories CRUD; default five categories on company create
- Expense create with manager/accounting routing + Frankfurter adapter
  (`adapter/frankfurter`)
- Pending queues; manager/accounting approve|reject (`reason` required on
  reject)
- Grant/revoke `accountant` via `employee_roles`
- Manager unassign requeues orphan `manager_approval` expenses

Schema: `migrations/finance/000001_create_expenses.up.sql`.

### Dashboard

Extended in `company/adapter/http/dashboard.go`: me widgets
`timesheet_current_week` + `wfh_today`; manager/hr/accountant pending
widgets; `GET /dashboard/accountant`; `flags.is_accountant`.

### RBAC

`migrations/core/000008_seed_rbac_step5.up.sql` seeds Step 5 permissions,
`accountant` role, and grants for admin/hr/employee/manager/accountant.

## Module wiring

- Enabled in `config/modules.yaml`: `time`, `finance`
- Blank-imported in `cmd/api/main.go`

## Verify

```bash
docker compose up -d postgres-go
cd api-go
go run ./cmd/migrate up
go run ./cmd/api
```

## Deferred

PTO, expense receipts.
