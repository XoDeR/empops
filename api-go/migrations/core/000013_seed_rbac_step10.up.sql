INSERT INTO permissions (id, name)
VALUES
    (gen_random_uuid(), 'pto.view'),
    (gen_random_uuid(), 'pto.manage'),
    (gen_random_uuid(), 'flows.manage'),
    (gen_random_uuid(), 'wiki.view'),
    (gen_random_uuid(), 'wiki.create'),
    (gen_random_uuid(), 'wiki.update'),
    (gen_random_uuid(), 'wiki.delete'),
    (gen_random_uuid(), 'ama.view'),
    (gen_random_uuid(), 'ama.manage'),
    (gen_random_uuid(), 'groups.view'),
    (gen_random_uuid(), 'groups.manage'),
    (gen_random_uuid(), 'billing.view')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.name = 'administrator'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
    'pto.view', 'pto.manage',
    'flows.manage',
    'wiki.view', 'wiki.create', 'wiki.update', 'wiki.delete',
    'ama.view', 'ama.manage',
    'groups.view', 'groups.manage'
)
WHERE r.name = 'hr'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
    'pto.view',
    'groups.view', 'groups.manage'
)
WHERE r.name = 'manager'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
    'pto.view',
    'wiki.view', 'wiki.create', 'wiki.update',
    'ama.view',
    'groups.view', 'groups.manage'
)
WHERE r.name = 'employee'
ON CONFLICT DO NOTHING;
