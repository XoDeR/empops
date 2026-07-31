# Step 7 — Recruit (ATS)

Documents what Step 7 implements in `api-laravel/`.

## Goal

Hiring pipeline + public careers page: stage templates, job openings,
candidates with CV upload, hire into the employee directory, and
employee CSV import.

## What's implemented

### Recruit (`Modules/Recruit`)

- Stage templates CRUD + ordered stages (Adminland)
- Job openings CRUD, publish toggle, sponsors
- Candidates bucketed `to_sort` / `selected` / `rejected`
- Stage pass/fail, notes, participants
- CV files via Media Library collection `cv`
- Hire → creates Employee (position + optional team), fulfills opening
- Public careers under `/api/v1/jobs` (no JWT): browse, apply, CV,
  complete/abandon incomplete applications

### Employee CSV

- `POST /api/v1/companies/{companyId}/employees/import` (multipart `file`)
- Columns: `email,first_name,last_name,hired_at?,position_id?`

### RBAC

`recruiting.view|create|update|delete|hire|manage_templates` — admin/HR
all. Opening sponsors may view candidates and process stages/notes
without HR perms (service-layer check).

## Verify

```bash
cd api-laravel
composer dump-autoload
php artisan migrate --force
php artisan db:seed --class=RolePermissionSeeder --force
php artisan serve --port=8000
```

Public: `GET http://localhost:8000/api/v1/jobs`

## Deferred

`candidate_stage_tasks`; auto-reject siblings on hire; invite email on
hire; onboarding/offboarding (Step 10).
