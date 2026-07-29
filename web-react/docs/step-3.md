# Step 3 — Media uploads + places

Documents what Step 3 implements in `web-react/`.

## Goal

SPA against Laravel or Go Step 3 API: chunked upload client, avatar/logo attach, employee addresses UI.

## What's implemented

### Upload client (`src/lib/upload/`)

- `ChunkedUploader` — init / chunk / complete against `/api/upload`
- Base URL: `VITE_UPLOAD_BASE_URL`, or derived from `VITE_API_BASE_URL` origin + `/api/upload`
- `resolveMediaUrl` for relative `avatar_url` / `logo_url` paths

### UI

- `ImageUploadField` — used for employee avatar (Profile tab) and company logo (Adminland)
- `PlacesSection` — list / create / edit / activate / delete addresses on Profile tab
- Types: `Place`, `Country`, `avatar_url`, `logo_url` on employee/company

### Flow

1. Select image → chunked upload → `{ media_id, temporary_upload_id }`
2. `PUT .../avatar` or `PUT .../logo` with those IDs
3. Invalidate React Query caches so URLs refresh

## Verify

```bash
cd web-react
npm run dev
# Point VITE_API_BASE_URL at Laravel (:8000) or Go (:8080) /api/v1
# Profile → upload avatar + add address; Adminland → upload logo
```
