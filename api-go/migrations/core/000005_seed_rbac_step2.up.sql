-- Step 2 RBAC: teams, hierarchy, manager role.

INSERT INTO permissions (id, name)
VALUES
    (gen_random_uuid(), 'teams.view'),
    (gen_random_uuid(), 'teams.create'),
    (gen_random_uuid(), 'teams.update'),
    (gen_random_uuid(), 'teams.delete'),
    (gen_random_uuid(), 'teams.manage_members'),
    (gen_random_uuid(), 'hierarchy.assign')
ON CONFLICT (name) DO NOTHING;

INSERT INTO roles (id, name)
VALUES (gen_random_uuid(), 'manager')
ON CONFLICT (name) DO NOTHING;

-- Administrator already has all permissions via prior seed; re-sync any new ones.
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
    'teams.view', 'teams.create', 'teams.update', 'teams.delete', 'teams.manage_members',
    'hierarchy.assign'
)
WHERE r.name = 'hr'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name IN ('teams.view')
WHERE r.name = 'employee'
ON CONFLICT DO NOTHING;
