# Step 6 — Projects

Documents what Step 6 implements in `api-laravel/`.

## Goal

Lightweight project management: projects with members, status updates,
links, messages, decisions, task lists, boards/issues/sprints, file
attachments, and timesheet entries linked to project tasks.

## What's implemented

### Project (`Modules/Project`)

- `GET|POST /api/v1/companies/{companyId}/projects`
- `GET|PATCH|DELETE .../projects/{projectId}`
- Members, lead, teams attach/detach
- Links CRUD; status updates list/create
- Messages CRUD + comments; decisions + deciders
- Task lists + tasks (toggle complete, comments, time-entry listing)
- Boards with backlog/active sprints; issues (create, reorder, assignees,
  story points); company `issue-types` (Bug/Story/Task/Epic seeded)
- Files via Media Library collection `files` (chunked upload attach)

### Time wiring (`Modules/Time`)

- `time_tracking_entries.project_id` / `project_task_id` (nullable)
- Upsert: ad-hoc one row per day when both FKs null; otherwise per
  `(day, project_task_id)`
- `GET .../timesheets/projects` and `.../projects/{projectId}/tasks`

### RBAC

`projects.view|create|update|delete|manage_members` — admin/HR all;
employee `view` + `create`. Nested writes also require membership (or
manage permission) in the service layer.

## Verify

```bash
cd api-laravel
composer dump-autoload
php artisan migrate --force
php artisan db:seed --class=RolePermissionSeeder --force
php artisan serve --port=8000
```

## Deferred

Member activity feed, issue parent/child graph, story-point history,
unread message/comment polish.
