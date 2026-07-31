# Step 6 — Projects

Documents what Step 6 implements in `web-react/`.

## Goal

SPA against Laravel or Go Step 6 API: project list/detail, board + tasks,
file attach, and timesheet logging against project tasks.

## What's implemented

- Types: `Project`, boards/sprints/issues, tasks, messages, decisions,
  files; `TimesheetEntry` project fields
- Nav: **Projects** in company layout
- `ProjectsPage` — list + create
- `ProjectDetailPage` tabs: Overview (status/members/links/updates),
  Messages, Decisions, Tasks, Board, Files (chunked upload attach)
- Dashboard timesheet widget: project + task pickers; shows linked
  project/task on entries; ad-hoc still works without pickers

## Verify

```bash
cd web-react
npm run dev
# Point VITE_API_BASE_URL at Laravel (:8000) or Go (:8080) /api/v1
# Projects → create → board/tasks/files; Me dashboard → log time to a task
```

## Deferred

Issue detail drawer polish, drag-and-drop board reorder UX, unread badges.
