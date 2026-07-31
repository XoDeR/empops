INSERT INTO permissions (id, name)
VALUES
    (gen_random_uuid(), 'morale.view'),
    (gen_random_uuid(), 'morale.log'),
    (gen_random_uuid(), 'one_on_ones.view'),
    (gen_random_uuid(), 'one_on_ones.manage'),
    (gen_random_uuid(), 'rate_manager.answer'),
    (gen_random_uuid(), 'rate_manager.view_results'),
    (gen_random_uuid(), 'skills.view'),
    (gen_random_uuid(), 'skills.manage'),
    (gen_random_uuid(), 'e_coffee.view'),
    (gen_random_uuid(), 'e_coffee.manage'),
    (gen_random_uuid(), 'discipline.view'),
    (gen_random_uuid(), 'discipline.manage')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.name = 'administrator'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
    'morale.view', 'morale.log',
    'one_on_ones.view', 'one_on_ones.manage',
    'rate_manager.answer', 'rate_manager.view_results',
    'skills.view', 'skills.manage',
    'e_coffee.view', 'e_coffee.manage',
    'discipline.view', 'discipline.manage'
)
WHERE r.name = 'hr'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
    'morale.view', 'morale.log',
    'one_on_ones.view', 'one_on_ones.manage',
    'rate_manager.view_results',
    'skills.view',
    'e_coffee.view',
    'discipline.view', 'discipline.manage'
)
WHERE r.name = 'manager'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
    'morale.log',
    'one_on_ones.view',
    'rate_manager.answer',
    'skills.view',
    'e_coffee.view'
)
WHERE r.name = 'employee'
ON CONFLICT DO NOTHING;
