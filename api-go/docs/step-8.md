# Step 8 — Grow (engagement & performance)

Documents what Step 8 implements in `api-go/`.

## Goal

Go parity with the Step 8 Grow contract: morale, one-on-ones,
rate-your-manager, skills, e-coffee, and discipline with file attach.

All company routes live under `/api/v1/companies/{companyId}`, use the
shared `pkg/response` envelope, and are guarded by `httpauth.RequireAuth`
+ `companyauth.RequireMember` (+ `RequirePermission` where noted).

## What's implemented

### Grow (`internal/modules/grow`)

- Daily morale + company/team history CLIs
- One-on-ones (talking points, action items, notes, mark happened)
- Rate-your-manager pending answers + surveys
- Skills directory + employee attach/detach
- e-Coffee flag, current match, mark happened
- Discipline cases/events + media attach (`discipline_event` /
  collection `discipline`)

Schema: `migrations/grow/000001_create_grow.up.sql`.

### RBAC

`migrations/core/000011_seed_rbac_step8.up.sql` seeds Grow permissions
and grants for administrator, hr, manager, and employee.

### Jobs

| Command | Purpose |
|---|---|
| `go run ./cmd/log-company-morale [date]` | Company morale snapshots |
| `go run ./cmd/log-team-morale [date]` | Team morale snapshots |
| `go run ./cmd/rate-manager-start` | Start monthly surveys |
| `go run ./cmd/rate-manager-stop [--force]` | Stop expired surveys |
| `go run ./cmd/e-coffee-start` | Pair companies with `e_coffee_enabled` |

### Dashboard widgets

Me / Manager / HR shells include Grow widgets (`morale_today`,
`one_on_ones_open`, `discipline_active`, etc.).

## Verify

```bash
cd api-go
go run ./cmd/migrate
go run ./cmd/api
# Listen :8080
```

## Deferred

OKRs; richer morale analytics; email digests for rate-your-manager.
