-- Core SQLC query sources: users, refresh_tokens.
-- Roles/permissions/employee_roles are queried from the employee module's
-- own SQL (sql/employee/queries.sql) since it owns the RBAC-check use case;
-- these tables still physically live in the shared Postgres database.

-- name: CreateUser :one
INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
VALUES ($1, $2, $3, $4, now(), now())
RETURNING id, email, name, password_hash, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, email, name, password_hash, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, name, password_hash, created_at, updated_at
FROM users
WHERE email = $1;

-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (id, user_id, jti, expires_at, created_at)
VALUES ($1, $2, $3, $4, now());

-- name: GetRefreshTokenByJTI :one
SELECT id, user_id, jti, expires_at, revoked_at, created_at
FROM refresh_tokens
WHERE jti = $1;

-- name: RevokeRefreshTokenByJTI :exec
UPDATE refresh_tokens
SET revoked_at = now()
WHERE jti = $1 AND revoked_at IS NULL;
