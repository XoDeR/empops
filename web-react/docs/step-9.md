# Step 9 — Hardware & software inventory

Documents what Step 9 implements in `web-react/`.

## Goal

SPA against Laravel or Go Step 9 API: hardware and software inventory
for HR/admin, plus assigned assets on the employee Work tab.

## What's implemented

- Types: `Hardware`, `Software`
- Pages: `HardwareListPage`, `HardwareDetailPage`,
  `SoftwareListPage`, `SoftwareDetailPage` (seats, FX display, files)
- Nav: Hardware + Software (HR/admin)
- Adminland: Hardware & software links
- Employee detail: assigned hardware / software (self or HR)

## Verify

```bash
cd web-react
npm install
npm run dev
# Point VITE_API_BASE_URL at Laravel (:8000) or Go (:8080) /api/v1
# Adminland → Manage hardware → create → lend
# Adminland → Manage software → create with purchase amount → assign seats
# Employees → open self/HR view → Work tab shows assigned assets
```

## Deferred

Hardware cost UI; license expiry reminders.
