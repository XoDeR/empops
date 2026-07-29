# Step 3 — Media uploads + places

Documents what Step 3 implements in `api-laravel/`.

## Goal

Own-server file uploads (no Uploadcare), employee avatar + company logo via Spatie Media Library, and employee address CRUD with a swappable geocoder.

## What's implemented

### Uploads (`Modules/Uploads`)

Chunked / stream API compatible with `file-uploads-go` upload-lib (base **`/api/upload`**, not `/api/v1`):

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/init` | Start chunked session |
| POST | `/chunk?upload_id=&chunk=` | Upload chunk |
| POST | `/complete?upload_id=` | Assemble file → Spatie media on `TemporaryUpload` |
| GET | `/status?upload_id=` | Resume status |
| POST | `/stream` | Multipart stream upload |
| GET | `/progress` | SSE progress (stream) |

Complete / stream responses include `media_id` and `temporary_upload_id`.

`MediaAttachService` moves media from a temporary upload onto an Employee/Company collection.

### Media attach

- `PUT /api/v1/companies/{companyId}/employees/{employeeId}/avatar` — body `{ temporary_upload_id, media_id }` (self or `employees.update`)
- `PUT /api/v1/companies/{companyId}/logo` — same body (`company.update`)
- Employee payload: `avatar_url`; company payload: `logo_url` (Spatie `getFirstMediaUrl`)

Requires `php artisan storage:link` for public disk URLs.

### Place (`Modules/Place`)

- Tables: `countries` (seeded), `places` (polymorphic on employee)
- Geocoder: `GEOCODER_DRIVER=noop|nominatim` (`Modules/Place/config/config.php`)
- Routes (auth + company member):

| Method | Path |
|--------|------|
| GET | `/api/v1/countries` |
| GET/POST | `/api/v1/companies/{companyId}/employees/{employeeId}/places` |
| PATCH/DELETE | `/api/v1/companies/{companyId}/places/{placeId}` |
| PUT | `/api/v1/companies/{companyId}/places/{placeId}/activate` |

Self-service for own places; HR uses `places.*` permissions for others.

### RBAC

`RolePermissionSeeder` adds: `places.view|create|update|delete`, `countries.view`.

Modules enabled in `modules_statuses.json`: `Uploads`, `Place`.

## Verify

```bash
cd api-laravel
composer dump-autoload
php artisan migrate --force
php artisan db:seed --class=RolePermissionSeeder --force
php artisan storage:link
php artisan serve --port=8000

# 1) Chunked upload → note media_id + temporary_upload_id
# 2) PUT .../avatar or .../logo with those IDs
# 3) Create/list places under an employee
```

## Deferred

Nominatim production rate limits / caching, image variants, multi-collection media UI.
