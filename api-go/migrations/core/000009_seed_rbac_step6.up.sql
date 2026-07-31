INSERT INTO permissions (id, name)
VALUES
    (gen_random_uuid(), 'projects.view'),
    (gen_random_uuid(), 'projects.create'),
    (gen_random_uuid(), 'projects.update'),
    (gen_random_uuid(), 'projects.delete'),
    (gen_random_uuid(), 'projects.manage_members')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.name = 'administrator'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
    'projects.view', 'projects.create', 'projects.update', 'projects.delete', 'projects.manage_members'
)
WHERE r.name = 'hr'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN ('projects.view', 'projects.create')
WHERE r.name = 'employee'
ON CONFLICT DO NOTHING;
