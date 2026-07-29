-- Step 3 RBAC: places and countries.

INSERT INTO permissions (id, name)
VALUES
    (gen_random_uuid(), 'places.view'),
    (gen_random_uuid(), 'places.create'),
    (gen_random_uuid(), 'places.update'),
    (gen_random_uuid(), 'places.delete'),
    (gen_random_uuid(), 'countries.view')
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
    'places.view', 'places.create', 'places.update', 'places.delete', 'countries.view'
)
WHERE r.name = 'hr'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name IN ('places.view', 'countries.view')
WHERE r.name = 'employee'
ON CONFLICT DO NOTHING;
