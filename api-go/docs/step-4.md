# Step 4 — Communicate (core collaboration)

Documents what Step 4 implements in `api-go/`.

## Goal

Go parity with the Step 4 "Communicate" contract: worklogs, company/team
news, ships, in-app notifications, and a lightweight Q&A (questions +
answers). Guess-the-employee, Wiki, and AMA are explicitly out of scope.

All new routes live under the existing `/api/v1/companies/{companyId}`
prefix, use the shared `pkg/response` envelope, and are guarded by
`httpauth.RequireAuth` + `companyauth.RequireMember` (+ `RequirePermission`
where noted).

## What's implemented

### Worklogs (extends `internal/modules/employee`)

- `POST /companies/{companyId}/worklogs` — self only. Body:
  `{ content, logged_on? }` (`logged_on` defaults to today, `YYYY-MM-DD`).
  One worklog per employee/day (`UNIQUE (employee_id, logged_on)`; duplicate
  → `409 Conflict`). On success, resets `employees.consecutive_worklog_missed`
  to `0`.
- `GET /companies/{companyId}/employees/{employeeId}/worklogs?from=&to=` —
  self, the employee's manager (via `direct_reports`), or `worklogs.view`.
- `DELETE /companies/{companyId}/employees/{employeeId}/worklogs/{worklogId}`
  — self, manager, or `worklogs.delete`.
- `GET /companies/{companyId}/teams/{teamId}/worklogs?date=` — team member or
  `worklogs.view`. Defaults `date` to today; returns each team member's
  worklog (with a nested `employee` summary) logged on that date.

Schema: `migrations/employee/000003_add_worklogs.up.sql` — adds
`employees.consecutive_worklog_missed` (int, default 0) and the `worklogs`
table.

### Company news (extends `internal/modules/company`)

- `GET /companies/{companyId}/news`, `GET .../news/{newsId}` — any member.
- `POST /companies/{companyId}/news` — `news.create`. Body:
  `{ title, content }`. Snapshots `author_name` from the creator at write
  time (matches the `author_id` nullable-on-delete pattern).
- `PATCH .../news/{newsId}` — `news.update`. `DELETE .../news/{newsId}` —
  `news.delete`.

Schema: `migrations/company/000003_create_company_news_and_questions.up.sql`
— `company_news`, `questions`, `answers`.

### Team news (extends `internal/modules/team`)

- `GET/POST /companies/{companyId}/teams/{teamId}/news` — team member (via
  `employee_team`) or `team-news.view` / `team-news.create`.
- `GET/PATCH/DELETE .../news/{newsId}` — read same as list; update/delete
  require the author or `team-news.update` / `team-news.delete`.

Schema: `migrations/team/000002_create_team_news_and_ships.up.sql` —
`team_news`, `ships`, `employee_ship`.

### Ships (extends `internal/modules/team`)

- `GET/POST /companies/{companyId}/teams/{teamId}/ships` — team member or
  `ships.view` / `ships.create`. POST body:
  `{ title, description?, employee_ids?: uuid[] }`.
- `GET/DELETE .../ships/{shipId}` — read same as list; delete requires the
  author or `ships.delete`.
- Creating a ship with `employee_ids` inserts an
  `employee_attached_to_recent_ship` notification (via `pkg/notify`) for
  every attached employee except the actor, with
  `objects = { ship_title, team_id, ship_id, author_name }`.

### Notifications (new `internal/modules/notification`)

- `GET /companies/{companyId}/notifications` → `{ items: [...], unread_count }`
  for the current employee, newest first (capped at 100).
- `POST /companies/{companyId}/notifications/read` — body `{ ids?: uuid[] }`;
  an empty/missing list marks every unread notification as read for the
  actor, otherwise only the listed ids.
- `pkg/notify.Create(ctx, pool, companyID, employeeID, action, objects)` is
  the shared helper any module can call to insert a notification row (used
  today by the team module's ship-attachment flow).

Schema: `migrations/notification/000001_create_notifications.up.sql` —
`notifications` (`read_at IS NULL` = unread).

### Q&A (extends `internal/modules/company`)

- `GET /companies/{companyId}/questions`, `GET .../questions/active` (active
  question + `my_answer`, or `null` if none), `GET .../questions/{questionId}`
  (includes `answers` + `answer_count` + `my_answer`) — any member.
- `POST .../questions` — `questions.create`. `PATCH`/`DELETE
  .../questions/{questionId}` — `questions.update` / `questions.delete`.
- `PUT .../questions/{questionId}/activate` / `.../deactivate` —
  `questions.manage`. Activating one question deactivates any other active
  question in the company (transactional; also enforced by a partial unique
  index on `questions (company_id) WHERE active`).
- `POST .../questions/{questionId}/answers` — self; body `{ body }`; upserts
  on `(question_id, employee_id)` so re-submitting edits the existing answer.
- `PATCH`/`DELETE .../questions/{questionId}/answers/{answerId}` — self or
  `questions.update`.

### Dashboard widgets (extends `internal/modules/company` dashboard)

The `me` dashboard (`GET /companies/{companyId}/dashboard/me`) now returns
three widgets instead of an empty array:

```json
{
  "view": "me",
  "widgets": [
    { "type": "worklog_today", "data": { "logged": true, "worklog": { "id": "...", "content": "...", "logged_on": "2026-07-29" }, "consecutive_missed": 0 } },
    { "type": "active_question", "data": { "id": "...", "title": "...", "answered": false } },
    { "type": "unread_notifications", "data": { "count": 3 } }
  ],
  "flags": { "is_manager": false, "can_manage_hr": false, "is_admin": false }
}
```

`team` / `manager` / `hr` dashboards are unchanged (still empty `widgets`).

### Missed-worklogs job (`cmd/missed-worklogs`)

A small CLI that increments `consecutive_worklog_missed` for every unlocked
employee who has no worklog for a given date (defaults to today, UTC):

```bash
go run ./cmd/missed-worklogs               # today
go run ./cmd/missed-worklogs 2026-07-29    # backfill a specific date
```

Intended to run once daily near end-of-day via cron/systemd-timer/scheduled
task, e.g.:

```cron
# 23:55 UTC daily
55 23 * * * cd /path/to/api-go && EMPOPS_DB_DSN=postgres://... ./bin/missed-worklogs
```

Build it alongside the other binaries: `go build -o bin/missed-worklogs ./cmd/missed-worklogs`.

### RBAC

`migrations/core/000007_seed_rbac_step4.up.sql` seeds:

```
worklogs.view, worklogs.delete,
news.view, news.create, news.update, news.delete,
team-news.view, team-news.create, team-news.update, team-news.delete,
ships.view, ships.create, ships.delete,
questions.view, questions.create, questions.update, questions.delete, questions.manage
```

- `administrator` — every permission (cross-join, same pattern as prior
  steps).
- `hr` — every Step 4 permission listed above.
- `employee` — `worklogs.view`, `news.view`, `team-news.view`,
  `team-news.create`, `ships.view`, `ships.create`, `questions.view`.

Most read routes (news, team news, ships, questions) don't gate on these
permissions directly — company membership (or team membership, for team-
scoped resources) is enough, matching the "GET any member" / "team member"
wording in the contract. The seeded `*.view` permissions exist for API
consumers that want a permission-based capability check and as an escape
hatch for non-members (e.g. HR viewing a team they're not on).

## Module structure

Followed the existing modular-DDD conventions (Chi router, raw pgx SQL in
handlers, `pkg/response` envelope, `pkg/uuidv7` IDs):

- `internal/modules/employee/adapter/http/worklogs.go` — worklog handlers.
- `internal/modules/company/adapter/http/news.go` — company news handlers.
- `internal/modules/company/adapter/http/questions.go` — Q&A handlers.
- `internal/modules/company/adapter/http/dashboard.go` — `me` widgets added.
- `internal/modules/team/adapter/http/news.go` — team news handlers.
- `internal/modules/team/adapter/http/ships.go` — ship handlers.
- `internal/modules/notification/` — new module (`module.go`, `register.go`,
  `adapter/http/handler.go`), enabled in `config/modules.yaml` and
  blank-imported in `cmd/api/main.go`.
- `pkg/notify/notify.go` — shared notification-insert helper.

## Verify

```bash
docker compose up -d postgres-go
cd api-go
# set EMPOPS_DB_DSN if needed (see .env.example)
go run ./cmd/migrate up
go run ./cmd/api

# Then exercise worklogs / news / ships / notifications / questions under
# /api/v1/companies/{companyId}/...
```

## Deferred

Guess-the-employee, Wiki, AMA (explicitly out of scope for Step 4).
