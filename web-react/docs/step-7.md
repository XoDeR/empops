# Step 7 — Recruit (ATS)

Documents what Step 7 implements in `web-react/`.

## Goal

SPA against Laravel or Go Step 7 API: recruiting list/detail, Adminland
templates + CSV import, and public careers apply flow.

## What's implemented

- Types: `JobOpening`, `Candidate`, stage templates, public job shapes
- Nav: **Recruiting** (HR/admin) in company layout
- `RecruitingPage` — open/fulfilled openings + create
- `JobOpeningDetailPage` — candidate buckets, pass/fail, notes,
  participants, CV, hire form
- Adminland: stage templates CRUD; employee CSV upload
- Public routes: `/jobs`, `/jobs/:companySlug`,
  `/jobs/:companySlug/jobs/:jobSlug` (apply wizard with chunked CV)

## Verify

```bash
cd web-react
npm run dev
# Point VITE_API_BASE_URL at Laravel (:8000) or Go (:8080) /api/v1
# Adminland → templates → Recruiting → create opening → toggle active
# Open /jobs → apply → Recruiting → process stages → hire
# Adminland → CSV import employees
```

## Deferred

Drag-and-drop pipeline polish; sponsor-only nav entry without HR role.
