DELETE FROM role_permissions;
DELETE FROM roles WHERE name IN ('administrator', 'hr', 'employee');
DELETE FROM permissions WHERE name IN (
    'company.update',
    'adminland.access',
    'employees.view', 'employees.create', 'employees.update', 'employees.delete', 'employees.invite',
    'positions.view', 'positions.create', 'positions.update', 'positions.delete',
    'employee-statuses.view', 'employee-statuses.create', 'employee-statuses.update', 'employee-statuses.delete'
);
