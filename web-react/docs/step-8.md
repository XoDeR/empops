# Step 8 — Grow (engagement & performance)

Documents what Step 8 implements in `web-react/`.

## Goal

SPA against Laravel or Go Step 8 API: Me dashboard Grow widgets,
one-on-one detail, discipline cases, skills directory, morale chart,
and Adminland e-Coffee toggle.

## What's implemented

- Types: morale, one-on-ones, rate-your-manager, skills, e-coffee,
  discipline; dashboard widget variants
- Me dashboard: morale check-in, open 1:1s, rate-your-manager, e-coffee
- Manager/HR: open 1:1s summary, discipline count + links
- Pages: `OneOnOneDetailPage`, `DisciplineListPage`,
  `DisciplineDetailPage` (chunked file upload), `SkillsPage`,
  `MoraleHistoryPage` (recharts)
- Nav: Discipline (manager/HR), Skills + Morale (HR/admin)
- Adminland: e-Coffee enable toggle

## Verify

```bash
cd web-react
npm install
npm run dev
# Point VITE_API_BASE_URL at Laravel (:8000) or Go (:8080) /api/v1
# Me → log morale → open 1:1
# Adminland → enable e-Coffee → run e-coffee-start job
# Discipline → create case → add event → attach file
# Morale → view company history chart
```

## Deferred

OKRs; richer morale filters by team; drag-and-drop discipline polish.
