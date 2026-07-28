# Step 2 — Teams, hierarchy, and org structure

Documents what Step 2 implements in `web-react/`.

## Goal

SPA against Laravel (or Go) Step 2 API: teams, employee Work tab (manager + teams), Me/Team/Manager/HR dashboard shells.

## What's implemented

### Navigation (`CompanyLayout`)

- Dashboard, Employees, Teams; Adminland still HR/Admin only

### Routes

- `/companies/:id/dashboard/:view` — Me / Team / Manager / HR shells
- `/companies/:id/teams` + `/teams/:teamId` — list/detail, members, lead
- `/companies/:id/employees/:employeeId` — Profile + Work tabs (manager assign, teams, reports)

### Types

Extended in `src/types/api.ts`: `Team`, `EmployeeSummary`, `DashboardShell`, hierarchy fields on `Employee`.

## Verify

```bash
cd web-react
npm run dev
# Register → company → create team → assign lead/member → assign manager → open Work tab + Manager dashboard
```
