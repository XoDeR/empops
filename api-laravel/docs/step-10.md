# Step 10 — PTO, flows, knowledge, groups, and billing

Documents what Step 10 implements in `api-laravel/`.

## Goal

Complete the employee lifecycle features around paid time off, onboarding and
offboarding flows, internal knowledge, AMA sessions, working groups, and
optional paid-plan usage tracking.

## What's implemented

### PTO (`Modules/Time`)

- Annual company PTO policies and generated weekday/weekend calendars
- Calendar day overrides with worked-day recalculation
- Employee holiday allowances, balances, and planned holidays
- Idempotent daily accrual through `empops:calculate-timeoff {date?}`
- New employees inherit the current policy defaults

### Flows, wiki, and AMA (`Modules/Company`)

- Join/leave flows containing relative steps and notification actions
- Automatic flow scheduling after a candidate is hired or an employee is locked
- Due action processing through `empops:process-flows {date?}`
- Company wikis, pages, page revisions, and page-view tracking
- AMA sessions with one active session per company, anonymous questions, and
  answered status

### Groups (`Modules/Group`)

- Company groups with members and a mission
- Meetings, happened status, attendance, and guests
- Ordered agenda items, presenters, completion status, and decisions
- Group members can manage their groups; `groups.manage` can manage every group

### Billing (`Modules/Billing`)

- Daily active-employee usage snapshots and employee detail rows
- Monthly invoices based on the month's maximum usage snapshot
- `empops:log-usage` and `empops:create-invoices`
- Billing routes and schedules are enabled only when
  `empops.enable_paid_plan` is true

### Instance configuration

- Public `GET /api/v1/instance` endpoint exposes signup, demo, and paid-plan flags
- Registration returns 403 when `empops.enable_signups` is false

## Scheduled jobs

- PTO accrual daily at 23:00
- Flow actions daily at 23:05
- Paid-plan usage daily at 23:10
- Paid-plan invoices on the first day of each month

## Verify

```bash
cd api-laravel
php artisan migrate --force
php artisan db:seed --class=RolePermissionSeeder --force
php artisan route:list --path=api/v1
php artisan empops:calculate-timeoff
php artisan empops:process-flows
```
