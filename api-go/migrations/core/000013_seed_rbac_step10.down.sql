DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN (
        'pto.view', 'pto.manage',
        'flows.manage',
        'wiki.view', 'wiki.create', 'wiki.update', 'wiki.delete',
        'ama.view', 'ama.manage',
        'groups.view', 'groups.manage',
        'billing.view'
    )
);

DELETE FROM permissions WHERE name IN (
    'pto.view', 'pto.manage',
    'flows.manage',
    'wiki.view', 'wiki.create', 'wiki.update', 'wiki.delete',
    'ama.view', 'ama.manage',
    'groups.view', 'groups.manage',
    'billing.view'
);
