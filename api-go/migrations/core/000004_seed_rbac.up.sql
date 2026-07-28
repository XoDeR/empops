-- Seeds the same roles/permissions as Laravel's RolePermissionSeeder so both
-- backends share row-compatible RBAC data when pointed at compatible DBs.

INSERT INTO permissions (id, name)
VALUES
    (gen_random_uuid(), 'company.update'),
    (gen_random_uuid(), 'adminland.access'),
    (gen_random_uuid(), 'employees.view'),
    (gen_random_uuid(), 'employees.create'),
    (gen_random_uuid(), 'employees.update'),
    (gen_random_uuid(), 'employees.delete'),
    (gen_random_uuid(), 'employees.invite'),
    (gen_random_uuid(), 'positions.view'),
    (gen_random_uuid(), 'positions.create'),
    (gen_random_uuid(), 'positions.update'),
    (gen_random_uuid(), 'positions.delete'),
    (gen_random_uuid(), 'employee-statuses.view'),
    (gen_random_uuid(), 'employee-statuses.create'),
    (gen_random_uuid(), 'employee-statuses.update'),
    (gen_random_uuid(), 'employee-statuses.delete')
ON CONFLICT (name) DO NOTHING;

INSERT INTO roles (id, name)
VALUES
    (gen_random_uuid(), 'administrator'),
    (gen_random_uuid(), 'hr'),
    (gen_random_uuid(), 'employee')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'administrator'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name IN (
    'adminland.access',
    'employees.view', 'employees.create', 'employees.update', 'employees.delete', 'employees.invite',
    'positions.view', 'positions.create', 'positions.update', 'positions.delete',
    'employee-statuses.view', 'employee-statuses.create', 'employee-statuses.update', 'employee-statuses.delete'
)
WHERE r.name = 'hr'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name IN ('employees.view', 'positions.view', 'employee-statuses.view')
WHERE r.name = 'employee'
ON CONFLICT DO NOTHING;
