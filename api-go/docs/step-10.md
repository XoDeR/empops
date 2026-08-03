# Step 10 — PTO, company knowledge, groups & billing

Documents what Step 10 implements in `api-go/`.

## Goal

EmpOps APIs for paid time off, employee lifecycle flows, company wikis,
ask-me-anything sessions, groups and meetings, and optional billing.

All company routes live under `/api/v1/companies/{companyId}`, use the
shared `pkg/response` envelope, and are guarded by `httpauth.RequireAuth`
+ `companyauth.RequireMember` (+ `RequirePermission` where noted).

## What's implemented

### PTO (`internal/modules/time`)

- Company PTO policy CRUD and generated yearly work calendars
- Per-day work calendar overrides
- Employee holiday balances and planned holiday management
- `cmd/calculate-timeoff` daily accrual command
- Current-year policy defaults are copied when employees are created

### Flows, wiki and AMA (`internal/modules/company`)

- Join/leave flows with ordered steps and recipient actions
- Join runs scheduled after recruitment hire and leave runs after lock
- `cmd/process-flows` processes due action runs
- Wiki/page CRUD with automatic page revisions and page-view counts
- AMA session and question management, including anonymous questions

### Groups (`internal/modules/group`)

- Group CRUD and membership
- Meetings, attendees, agenda items, and decisions

### Billing (`internal/modules/billing`)

- Invoice listing guarded by `billing.view`
- Routes are mounted only when `ENABLE_PAID_PLAN=true`

### Instance features

`GET /api/v1/instance` is public and reports:

- `enable_signups` (`ENABLE_SIGNUPS`, default `true`)
- `demo_mode` (`DEMO_MODE`, default `false`)
- `enable_paid_plan` (`ENABLE_PAID_PLAN`, default `false`)

Registration is rejected when signups are disabled.

Schema: `migrations/time/000002_create_step10_tables.up.sql`.
RBAC: `migrations/core/000013_seed_rbac_step10.up.sql`.

## Verify

```bash
cd api-go
go run ./cmd/migrate
go test ./...
go run ./cmd/api
# GET /api/v1/instance
# GET /api/v1/companies/{id}/pto-policies
# GET /api/v1/companies/{id}/groups
```

Daily commands:

```bash
go run ./cmd/calculate-timeoff
go run ./cmd/process-flows
# When ENABLE_PAID_PLAN=true:
go run ./cmd/log-usage
go run ./cmd/create-invoices
```

## Deferred

Payment-provider integration, invoice delivery, and action-specific
email/notification delivery.
