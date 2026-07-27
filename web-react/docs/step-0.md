# Step 0 — React web shell

Documents what Step 0 implements in `web-react/`.

## Goal

Decoupled SPA shell that talks to the Laravel (or Go) API health and stub JWT auth endpoints.

## What's implemented

- Vite + React 19 + TypeScript
- TanStack Query for server state (`/health`, `/auth/me`)
- Zustand (+ persist) for access/refresh tokens and user
- React Hook Form + Zod login form
- Tailwind CSS v4 baseline (shadcn-ready styling primitives; full shadcn component kit lands with UI work)
- React Router wired in `main.tsx` (single shell route for Step 0)
- Bearer JWT HTTP client: `src/lib/api.ts` → `VITE_API_BASE_URL` (default `http://localhost:8000/api/v1`)

### UI

- Brand + health panel
- Stub login (any password accepted by API stubs)
- Signed-in panel calling `GET /auth/me`

## Env

```bash
cp .env.example .env
# optional: point at Go instead
# VITE_API_BASE_URL=http://localhost:8080/api/v1
```

## Verify

```bash
cd web-react
npm install
npm run dev
```

With Laravel on `:8000` (or Go on `:8080`), open http://localhost:5173 — health should show `ok`, login should store JWT and load `/auth/me`.

## Deferred

- Full shadcn/ui component library install
- Refresh-token rotation interceptor
- Tenancy / company switcher
- Feature routes beyond the shell
