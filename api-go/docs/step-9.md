# Step 9 — Hardware & software inventory

Documents what Step 9 implements in `api-go/`.

## Goal

Go parity with the Step 9 Hardware contract: hardware lend/regain,
software seats + FX, and license file attach.

All company routes live under `/api/v1/companies/{companyId}`, use the
shared `pkg/response` envelope, and are guarded by `httpauth.RequireAuth`
+ `companyauth.RequireMember` (+ `RequirePermission` where noted).

## What's implemented

### Hardware (`internal/modules/hardware`)

- Hardware CRUD, lend, regain, list filters, employee assigned list
- Software CRUD, seats give/revoke/all, employees-without, files
- FX via Finance Frankfurter client on create/update (soft-fail leaves
  converted fields null)

Schema: `migrations/hardware/000001_create_hardware.up.sql`.

### RBAC

`migrations/core/000012_seed_rbac_step9.up.sql` seeds Hardware/Software
permissions for administrator and hr.

## Verify

```bash
cd api-go
go run ./cmd/migrate
go run ./cmd/api
# Listen :8080
# POST /api/v1/companies/{id}/hardware
# POST /api/v1/companies/{id}/softwares
```

## Deferred

Hardware purchase/cost tracking; license expiry cron; audit log UI.
