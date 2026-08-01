INSERT INTO permissions (id, name)
VALUES
    (gen_random_uuid(), 'hardware.view'),
    (gen_random_uuid(), 'hardware.manage'),
    (gen_random_uuid(), 'software.view'),
    (gen_random_uuid(), 'software.manage')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.name = 'administrator'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
    'hardware.view', 'hardware.manage',
    'software.view', 'software.manage'
)
WHERE r.name = 'hr'
ON CONFLICT DO NOTHING;
