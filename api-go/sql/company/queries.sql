-- Company module SQLC query sources.

-- name: CreateCompany :one
INSERT INTO companies (id, name, slug, currency, code_to_join_company, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, now(), now())
RETURNING id, name, slug, currency, code_to_join_company, created_at, updated_at;

-- name: GetCompanyByID :one
SELECT id, name, slug, currency, code_to_join_company, created_at, updated_at
FROM companies
WHERE id = $1;

-- name: GetCompanyByJoinCode :one
SELECT id, name, slug, currency, code_to_join_company, created_at, updated_at
FROM companies
WHERE code_to_join_company = $1;

-- name: ExistsCompanyBySlug :one
SELECT EXISTS(SELECT 1 FROM companies WHERE slug = $1) AS exists;

-- name: ExistsCompanyBySlugExceptID :one
SELECT EXISTS(SELECT 1 FROM companies WHERE slug = $1 AND id != $2) AS exists;

-- name: ExistsCompanyByJoinCode :one
SELECT EXISTS(SELECT 1 FROM companies WHERE code_to_join_company = $1) AS exists;

-- name: UpdateCompany :one
UPDATE companies
SET name = $2, slug = $3, currency = $4, updated_at = now()
WHERE id = $1
RETURNING id, name, slug, currency, code_to_join_company, created_at, updated_at;
