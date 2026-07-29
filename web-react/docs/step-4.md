# Step 4 — Communicate (core collaboration)

Documents what Step 4 implements in `web-react/`.

## Goal

SPA against Laravel or Go Step 4 API: Me dashboard worklog/Q&A widgets,
notification bell, team news/ships, Adminland company news + questions.

## What's implemented

- Types: `Worklog`, `CompanyNews`, `TeamNews`, `Ship`, `AppNotification`,
  `Question`, `Answer`, typed `DashboardWidget`s
- Me dashboard: log today’s work, answer active question, unread count
- `NotificationBell` in company header (list + mark all read)
- Team detail: post/list/delete team news and ships (with employee attach)
- Adminland: company news + Q&A activate/deactivate

## Verify

```bash
cd web-react
npm run dev
# Point VITE_API_BASE_URL at Laravel (:8000) or Go (:8080) /api/v1
# Me dashboard → worklog + Q&A; Team → news/ships; Adminland → news/questions
```
