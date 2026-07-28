-- Reverse Step 2 RBAC seed (permissions remain; role_permissions for new perms removed).

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN (
        'teams.view', 'teams.create', 'teams.update', 'teams.delete', 'teams.manage_members',
        'hierarchy.assign'
    )
);

DELETE FROM employee_roles
WHERE role_id IN (SELECT id FROM roles WHERE name = 'manager');

DELETE FROM roles WHERE name = 'manager';

DELETE FROM permissions WHERE name IN (
    'teams.view', 'teams.create', 'teams.update', 'teams.delete', 'teams.manage_members',
    'hierarchy.assign'
);
