DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN (
        'timesheets.view', 'timesheets.approve', 'expenses.view', 'expenses.delete',
        'expenses.manage_categories', 'expenses.finalize'
    )
);
DELETE FROM permissions WHERE name IN (
    'timesheets.view', 'timesheets.approve', 'expenses.view', 'expenses.delete',
    'expenses.manage_categories', 'expenses.finalize'
);
DELETE FROM roles WHERE name = 'accountant';
