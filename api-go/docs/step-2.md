# Step 2 — Teams, hierarchy, and org structure

Documents what Step 2 implements in `api-go/`.

## Goal

Go parity with Laravel Step 2: teams, direct reports, manager role, dashboard shells, activity logs.

## What's implemented

### Team module (`internal/modules/team`)

- Enabled in `config/modules.yaml`
- Migrations: `migrations/team/`
- Same routes/permissions as Laravel

### Hierarchy

- Migration: `migrations/employee/000002_create_direct_reports`
- Handlers on employee module; RBAC seed `migrations/core/000005_seed_rbac_step2`

### Audit + dashboards

- `activity_logs` table (`migrations/company/000002_create_activity_logs`)
- Dashboard + audit-log routes on company module
- `pkg/audit` helper used by team/hierarchy mutations

## Verify

```bash
docker compose up -d postgres-go
cd api-go
go run ./cmd/migrate
go run ./cmd/api
```

## Deferred

Same as Laravel — news/ships/links and real dashboard widgets.
