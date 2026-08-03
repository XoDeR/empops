# Step 10 — Company operations

Documents what Step 10 implements in `web-react/`.

## Goal

SPA against the Laravel or Go Step 10 API for PTO, flows, shared knowledge,
AMA sessions, groups and meetings, billing visibility, and instance flags.

## What's implemented

- Types for all Step 10 API resources
- Wiki list and detail pages with page editing and revision history
- AMA sessions with member questions and HR answer status
- Groups with members, meetings, agenda items, and decisions
- Adminland sections for PTO policies, calendars, flows, and paid-plan invoices
- Employee Work tab holiday balance and planned holidays
- Public signup gating and a demo-mode banner from `/instance`
- Wiki, AMA, and Groups navigation for every company member

## Verify

```bash
cd web-react
npm install
npm run build
# Point VITE_API_BASE_URL at the Laravel or Go /api/v1 endpoint.
# Check Wiki, AMA, and Groups as an employee.
# Check PTO, flows, and conditional invoices in Adminland as HR.
# Check holiday balance and planning on an employee Work tab.
```

## Deferred

Markdown rendering, revision restoration, calendar bulk actions, and flow runs.
