# Step 5 — Operate: timesheets + expenses

Documents what Step 5 implements in `web-react/`.

## Goal

SPA against Laravel or Go Step 5 API: weekly timesheet editor + submit,
WFH toggle, expense submit, manager/HR/accountant approval queues, and
Adminland categories / WFH setting / accountants.

## What's implemented

- Types: `Timesheet`, `TimesheetEntry`, `Expense`, `ExpenseCategory`,
  extended `DashboardWidget` / `DashboardShell` (incl. `accountant`)
- Me dashboard: timesheet week editor + submit; WFH today; expense form
- Manager: pending timesheets approve/reject; pending expenses
- HR: orphan/past-week timesheets
- Accountant dashboard view (nav when accountant or HR/admin)
- Adminland: expense categories, WFH company toggle, grant/revoke
  accountants (`OperateAdminSections`)
- Company layout: `isAccountant` from membership roles

## Verify

```bash
cd web-react
npm run dev
# Point VITE_API_BASE_URL at Laravel (:8000) or Go (:8080) /api/v1
# Me → timesheet + expense; Manager/Accountant → queues; Adminland → categories
```

## Deferred

Project/task time entry fields, PTO UI, receipt uploads.
