# Step 3 — Media uploads + places

Documents what Step 3 implements in `api-go/`.

## Goal

Go parity with Laravel Step 3: chunked uploads on own disk, avatar/logo attach, places CRUD, noop geocoder.

## What's implemented

### Upload routes (`cmd/api`)

Mounted at **`/api/upload`** (same contract as Laravel / `file-uploads-go`):

- `POST /init`, `/chunk`, `/complete` (wrapped to register DB rows), `/stream`
- `GET /status`, `/progress`
- Config: `EMPOPS_UPLOAD_DIR` (default `./uploads`), `EMPOPS_UPLOAD_MAX_SIZE_BYTES`

Complete response includes `media_id` and `temporary_upload_id`.

### Media module (`internal/modules/media`)

- Migrations: `migrations/media/` (`temporary_uploads`, `media`)
- `PUT /api/v1/companies/{companyId}/employees/{employeeId}/avatar`
- `PUT /api/v1/companies/{companyId}/logo` (`company.update`)
- `GET /api/v1/media/{mediaId}/file` — serve file from upload dir
- Payloads: `avatar_url`, `logo_url` via `pkg/mediaurl`

### Place module (`internal/modules/place`)

- Migrations: `migrations/place/` (countries + places)
- Geocoder: `pkg/geocoder` (noop; Nominatim can be added later)
- Same routes as Laravel under `/api/v1`
- Enabled in `config/modules.yaml`; blank-imported in `cmd/api/main.go`

### RBAC

`migrations/core/000006_seed_rbac_step3` — places + countries permissions.

## Verify

```bash
docker compose up -d postgres-go
cd api-go
# set EMPOPS_DB_DSN if needed (see .env.example)
go run ./cmd/migrate up
go run ./cmd/api

# Upload base: http://127.0.0.1:8080/api/upload
# Then attach avatar/logo and exercise places CRUD
```

## Deferred

Nominatim geocoder wiring, stream-upload DB registration parity, image processing.
