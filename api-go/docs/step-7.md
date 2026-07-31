# Step 7 — Recruit (ATS)

Documents what Step 7 implements in `api-go/`.

## Goal

Go parity with the Step 7 Recruit contract: templates, openings,
candidate pipeline, public careers, CV attach, hire→employee, and
employee CSV import.

All company routes live under `/api/v1/companies/{companyId}`, use the
shared `pkg/response` envelope, and are guarded by `httpauth.RequireAuth`
+ `companyauth.RequireMember` (+ `RequirePermission` where noted).
Public careers routes under `/api/v1/jobs` have no auth.

## What's implemented

### Recruit (`internal/modules/recruit`)

- Stage templates + stages
- Job openings CRUD, toggle, sponsors
- Candidates (buckets), stage process, notes, participants
- Hire creates employee + optional team membership; fulfills opening
- Files: attach from temporary upload (`model_type=candidate`,
  collection `cv`)
- Public `/jobs` browse + apply flow (slug/uuid keys)

Schema: `migrations/recruit/000001_create_recruit.up.sql`.

### Employee CSV

- `POST /companies/{companyId}/employees/import` multipart `file`

### RBAC

`migrations/core/000010_seed_rbac_step7.up.sql` seeds `recruiting.*`
and grants for administrator and hr.

## Verify

```bash
cd api-go
go run ./cmd/migrate
go run ./cmd/api
# Listen :8080
```

## Deferred

Same as Laravel Step 7: stage tasks, auto-reject on hire, invite on
hire, onboarding flows.
