DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN (
        'projects.view', 'projects.create', 'projects.update', 'projects.delete', 'projects.manage_members'
    )
);

DELETE FROM permissions WHERE name IN (
    'projects.view', 'projects.create', 'projects.update', 'projects.delete', 'projects.manage_members'
);
