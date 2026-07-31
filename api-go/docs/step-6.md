# Step 6 — Projects

Documents what Step 6 implements in `api-go/`.

## Goal

Go parity with the Step 6 Projects contract: projects, collaboration,
boards/issues/sprints, media file attach, and timesheet task linkage.

All new routes live under `/api/v1/companies/{companyId}`, use the shared
`pkg/response` envelope, and are guarded by `httpauth.RequireAuth` +
`companyauth.RequireMember` (+ `RequirePermission` where noted).

## What's implemented

### Project (`internal/modules/project`)

- Projects CRUD; members/lead/teams; links; status updates
- Messages + comments; decisions + deciders
- Task lists/tasks (toggle, comments, time entries)
- Boards (creates Backlog + Sprint 1); sprint start/toggle; issues
  (key/`id_in_project`, assignees, points); issue types seeded per company
- Files: multi-file attach from temporary upload (`model_type=project`,
  collection `files`)

Schema: `migrations/project/000001_create_projects.up.sql`,
`migrations/project/000002_wire_timesheet_entries.up.sql`.

### Time (`internal/modules/time`)

- Entry upsert accepts optional `project_id` / `project_task_id`
- Partial unique indexes for ad-hoc vs task rows
- `GET /timesheets/projects` and `GET /timesheets/projects/{projectId}/tasks`

### RBAC

`migrations/core/000009_seed_rbac_step6.up.sql` seeds `projects.*` and
grants for administrator, hr, and employee.

## Verify

```bash
cd api-go
go run ./cmd/migrate
go run ./cmd/api
# Listen :8080
```

## Deferred

Same as Laravel Step 6: activity feed, issue hierarchy, unread polish.
