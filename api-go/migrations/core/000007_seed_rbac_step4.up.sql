-- Step 4 RBAC: worklogs, company/team news, ships, questions & answers.

INSERT INTO permissions (id, name)
VALUES
    (gen_random_uuid(), 'worklogs.view'),
    (gen_random_uuid(), 'worklogs.delete'),
    (gen_random_uuid(), 'news.view'),
    (gen_random_uuid(), 'news.create'),
    (gen_random_uuid(), 'news.update'),
    (gen_random_uuid(), 'news.delete'),
    (gen_random_uuid(), 'team-news.view'),
    (gen_random_uuid(), 'team-news.create'),
    (gen_random_uuid(), 'team-news.update'),
    (gen_random_uuid(), 'team-news.delete'),
    (gen_random_uuid(), 'ships.view'),
    (gen_random_uuid(), 'ships.create'),
    (gen_random_uuid(), 'ships.delete'),
    (gen_random_uuid(), 'questions.view'),
    (gen_random_uuid(), 'questions.create'),
    (gen_random_uuid(), 'questions.update'),
    (gen_random_uuid(), 'questions.delete'),
    (gen_random_uuid(), 'questions.manage')
ON CONFLICT (name) DO NOTHING;

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
    'worklogs.view', 'worklogs.delete',
    'news.view', 'news.create', 'news.update', 'news.delete',
    'team-news.view', 'team-news.create', 'team-news.update', 'team-news.delete',
    'ships.view', 'ships.create', 'ships.delete',
    'questions.view', 'questions.create', 'questions.update', 'questions.delete', 'questions.manage'
)
WHERE r.name = 'hr'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name IN (
    'worklogs.view',
    'news.view',
    'team-news.view', 'team-news.create',
    'ships.view', 'ships.create',
    'questions.view'
)
WHERE r.name = 'employee'
ON CONFLICT DO NOTHING;
