# Step 4 — Communicate (core collaboration)

Documents what Step 4 implements in `api-laravel/`.

## Goal

Internal communication hub: daily worklogs (+ missed-log job), company/team
news, recent ships, in-app notifications, and Q&A. Guess-the-employee, Wiki,
and AMA are deferred.

## What's implemented

### Worklogs (`Modules/Employee`)

- `POST /api/v1/companies/{companyId}/worklogs` — self; `{ content, logged_on? }`
- `GET .../employees/{employeeId}/worklogs?from=&to=` — self, manager, or `worklogs.view`
- `DELETE .../employees/{employeeId}/worklogs/{worklogId}` — self, manager, or `worklogs.delete`
- `GET .../teams/{teamId}/worklogs?date=` — team member or `worklogs.view`
- `employees.consecutive_worklog_missed`; Artisan `empops:mark-missed-worklogs` scheduled daily 23:00

### Company news + Q&A (`Modules/Company`)

- News CRUD under `/news` (`news.create|update|delete` for writes)
- Questions CRUD + activate/deactivate (`questions.manage`)
- Answers upsert/update/delete on active questions

### Team news + ships (`Modules/Team`)

- `/teams/{teamId}/news` CRUD (author or `team-news.*`)
- `/teams/{teamId}/ships` create/list/show/delete; attach `employee_ids` → notifications

### Notifications (`Modules/Notification`)

- `GET /notifications` → `{ items, unread_count }`
- `POST /notifications/read` — `{ ids? }` (empty = all)

### Dashboard

`GET /dashboard/me` widgets: `worklog_today`, `active_question`, `unread_notifications`.

### RBAC

Seeder adds Step 4 permissions; HR gets full communicate manage; employees get
views + `team-news.create` + `ships.create`.

## Verify

```bash
cd api-laravel
composer dump-autoload
php artisan migrate --force
php artisan db:seed --class=RolePermissionSeeder --force
php artisan serve --port=8000
```

## Deferred

Guess-the-employee, Wiki, AMA.
