INSERT INTO permissions (id, name)
VALUES
    (gen_random_uuid(), 'timesheets.view'),
    (gen_random_uuid(), 'timesheets.approve'),
    (gen_random_uuid(), 'expenses.view'),
    (gen_random_uuid(), 'expenses.delete'),
    (gen_random_uuid(), 'expenses.manage_categories'),
    (gen_random_uuid(), 'expenses.finalize')
ON CONFLICT (name) DO NOTHING;

INSERT INTO roles (id, name)
VALUES (gen_random_uuid(), 'accountant')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.name = 'administrator'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
    'timesheets.view', 'timesheets.approve', 'expenses.view', 'expenses.delete',
    'expenses.manage_categories', 'expenses.finalize'
)
WHERE r.name = 'hr'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN ('timesheets.view', 'expenses.view')
WHERE r.name = 'employee'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN ('expenses.view', 'expenses.finalize')
WHERE r.name = 'accountant'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name = 'timesheets.approve'
WHERE r.name = 'manager'
ON CONFLICT DO NOTHING;
