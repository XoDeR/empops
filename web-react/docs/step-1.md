# Step 1 — Auth + RBAC + Companies + Employees

Documents what Step 1 implements in `web-react/`.

## Goal

SPA against Laravel (or Go) Step 1 API: register/login, company switcher, employee directory, Adminland for HR/Admin.

## What's implemented

### Auth

- `/login`, `/register` pages (RHF + Zod)
- `src/lib/authFetch.ts` — Bearer token + refresh-on-401 interceptor
- Zustand persist for access/refresh/user (`src/stores/auth.ts`)

### Companies

- `/` — list, create (default currency EUR), join by code
- Company context via `src/stores/company.ts` + `CompanyLayout`

### Company routes (`/companies/:companyId/...`)

- `/employees` — directory; HR/Admin create, invite, edit; self edit name
- `/adminland` — company settings (admin), positions, employee statuses (HR/Admin)
- Adminland hidden/redirected for plain `employee` role

### Env

```bash
cp .env.example .env
# Laravel (default)
VITE_API_BASE_URL=http://localhost:8000/api/v1
# Go parity testing
# VITE_API_BASE_URL=http://localhost:8080/api/v1
```

## Verify

```bash
cd web-react
npm install
npm run dev
# Register → create company → add employee → confirm Adminland blocked for employee role
```
