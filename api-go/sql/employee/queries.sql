-- Employee module SQLC query sources: positions, employee_statuses,
-- employees, employee_roles (+ read-only lookups against the shared
-- companies/roles/permissions tables the employee module needs).

-- name: CompanyExists :one
SELECT EXISTS(SELECT 1 FROM companies WHERE id = $1) AS exists;

-- name: CreatePosition :one
INSERT INTO positions (id, company_id, title, created_at, updated_at)
VALUES ($1, $2, $3, now(), now())
RETURNING id, company_id, title, created_at, updated_at;

-- name: ListPositionsByCompany :many
SELECT id, company_id, title, created_at, updated_at
FROM positions
WHERE company_id = $1
ORDER BY title;

-- name: GetPositionByIDInCompany :one
SELECT id, company_id, title, created_at, updated_at
FROM positions
WHERE id = $1 AND company_id = $2;

-- name: UpdatePosition :one
UPDATE positions
SET title = $3, updated_at = now()
WHERE id = $1 AND company_id = $2
RETURNING id, company_id, title, created_at, updated_at;

-- name: DeletePosition :exec
DELETE FROM positions WHERE id = $1 AND company_id = $2;

-- name: CreateEmployeeStatus :one
INSERT INTO employee_statuses (id, company_id, name, type, created_at, updated_at)
VALUES ($1, $2, $3, $4, now(), now())
RETURNING id, company_id, name, type, created_at, updated_at;

-- name: ListEmployeeStatusesByCompany :many
SELECT id, company_id, name, type, created_at, updated_at
FROM employee_statuses
WHERE company_id = $1
ORDER BY name;

-- name: GetEmployeeStatusByIDInCompany :one
SELECT id, company_id, name, type, created_at, updated_at
FROM employee_statuses
WHERE id = $1 AND company_id = $2;

-- name: UpdateEmployeeStatus :one
UPDATE employee_statuses
SET name = $3, type = $4, updated_at = now()
WHERE id = $1 AND company_id = $2
RETURNING id, company_id, name, type, created_at, updated_at;

-- name: DeleteEmployeeStatus :exec
DELETE FROM employee_statuses WHERE id = $1 AND company_id = $2;

-- name: CreateEmployee :one
INSERT INTO employees (
    id, company_id, user_id, email, first_name, last_name, hired_at,
    position_id, employee_status_id, locked, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), now())
RETURNING id, company_id, user_id, email, first_name, last_name, hired_at,
    position_id, employee_status_id, invitation_link, invitation_used_at,
    locked, created_at, updated_at;

-- name: GetEmployeeByIDInCompany :one
SELECT id, company_id, user_id, email, first_name, last_name, hired_at,
    position_id, employee_status_id, invitation_link, invitation_used_at,
    locked, created_at, updated_at
FROM employees
WHERE id = $1 AND company_id = $2;

-- name: GetEmployeeByCompanyAndUser :one
SELECT id, company_id, user_id, email, first_name, last_name, hired_at,
    position_id, employee_status_id, invitation_link, invitation_used_at,
    locked, created_at, updated_at
FROM employees
WHERE company_id = $1 AND user_id = $2;

-- name: GetUnclaimedEmployeeByCompanyAndEmail :one
SELECT id, company_id, user_id, email, first_name, last_name, hired_at,
    position_id, employee_status_id, invitation_link, invitation_used_at,
    locked, created_at, updated_at
FROM employees
WHERE company_id = $1 AND email = $2 AND user_id IS NULL;

-- name: GetEmployeeByInvitationLink :one
SELECT id, company_id, user_id, email, first_name, last_name, hired_at,
    position_id, employee_status_id, invitation_link, invitation_used_at,
    locked, created_at, updated_at
FROM employees
WHERE invitation_link = $1;

-- name: ExistsEmployeeByCompanyAndUserExceptID :one
SELECT EXISTS(
    SELECT 1 FROM employees WHERE company_id = $1 AND user_id = $2 AND id != $3
) AS exists;

-- name: ListEmployeesByCompany :many
SELECT id, company_id, user_id, email, first_name, last_name, hired_at,
    position_id, employee_status_id, invitation_link, invitation_used_at,
    locked, created_at, updated_at
FROM employees
WHERE company_id = $1
ORDER BY last_name, first_name;

-- name: UpdateEmployee :one
UPDATE employees
SET email = $3, first_name = $4, last_name = $5, hired_at = $6,
    position_id = $7, employee_status_id = $8, locked = $9, updated_at = now()
WHERE id = $1 AND company_id = $2
RETURNING id, company_id, user_id, email, first_name, last_name, hired_at,
    position_id, employee_status_id, invitation_link, invitation_used_at,
    locked, created_at, updated_at;

-- name: SetEmployeeInvitation :one
UPDATE employees
SET invitation_link = $2, invitation_used_at = NULL, updated_at = now()
WHERE id = $1
RETURNING id, company_id, user_id, email, first_name, last_name, hired_at,
    position_id, employee_status_id, invitation_link, invitation_used_at,
    locked, created_at, updated_at;

-- name: ClaimEmployeeInvitation :one
UPDATE employees
SET user_id = $2, invitation_used_at = now(), updated_at = now()
WHERE id = $1
RETURNING id, company_id, user_id, email, first_name, last_name, hired_at,
    position_id, employee_status_id, invitation_link, invitation_used_at,
    locked, created_at, updated_at;

-- name: DeleteEmployee :exec
DELETE FROM employees WHERE id = $1 AND company_id = $2;

-- name: GetRoleIDByName :one
SELECT id FROM roles WHERE name = $1;

-- name: AssignEmployeeRole :exec
INSERT INTO employee_roles (employee_id, role_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ClearEmployeeRoles :exec
DELETE FROM employee_roles WHERE employee_id = $1;

-- name: ListEmployeeRoleNames :many
SELECT r.name
FROM employee_roles er
JOIN roles r ON r.id = er.role_id
WHERE er.employee_id = $1
ORDER BY r.name;

-- name: ListEmployeePermissionNames :many
SELECT DISTINCT p.name
FROM employee_roles er
JOIN role_permissions rp ON rp.role_id = er.role_id
JOIN permissions p ON p.id = rp.permission_id
WHERE er.employee_id = $1;
