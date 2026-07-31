INSERT INTO permissions (id, name)
VALUES
    (gen_random_uuid(), 'recruiting.view'),
    (gen_random_uuid(), 'recruiting.create'),
    (gen_random_uuid(), 'recruiting.update'),
    (gen_random_uuid(), 'recruiting.delete'),
    (gen_random_uuid(), 'recruiting.hire'),
    (gen_random_uuid(), 'recruiting.manage_templates')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.name = 'administrator'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
    'recruiting.view', 'recruiting.create', 'recruiting.update',
    'recruiting.delete', 'recruiting.hire', 'recruiting.manage_templates'
)
WHERE r.name = 'hr'
ON CONFLICT DO NOTHING;
