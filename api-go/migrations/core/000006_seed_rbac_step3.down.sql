DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN (
        'places.view', 'places.create', 'places.update', 'places.delete', 'countries.view'
    )
);

DELETE FROM permissions WHERE name IN (
    'places.view', 'places.create', 'places.update', 'places.delete', 'countries.view'
);
