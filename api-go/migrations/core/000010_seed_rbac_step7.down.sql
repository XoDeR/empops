DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name LIKE 'recruiting.%'
);

DELETE FROM permissions WHERE name LIKE 'recruiting.%';
