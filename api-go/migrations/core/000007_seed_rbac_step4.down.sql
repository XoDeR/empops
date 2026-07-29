DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN (
        'worklogs.view', 'worklogs.delete',
        'news.view', 'news.create', 'news.update', 'news.delete',
        'team-news.view', 'team-news.create', 'team-news.update', 'team-news.delete',
        'ships.view', 'ships.create', 'ships.delete',
        'questions.view', 'questions.create', 'questions.update', 'questions.delete', 'questions.manage'
    )
);

DELETE FROM permissions WHERE name IN (
    'worklogs.view', 'worklogs.delete',
    'news.view', 'news.create', 'news.update', 'news.delete',
    'team-news.view', 'team-news.create', 'team-news.update', 'team-news.delete',
    'ships.view', 'ships.create', 'ships.delete',
    'questions.view', 'questions.create', 'questions.update', 'questions.delete', 'questions.manage'
);
