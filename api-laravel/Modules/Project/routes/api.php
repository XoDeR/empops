<?php

use Illuminate\Support\Facades\Route;
use Modules\Auth\Http\Middleware\AuthenticateJwt;
use Modules\Company\Http\Middleware\EnsureCompanyMember;
use Modules\Company\Http\Middleware\EnsurePermission;
use Modules\Project\Http\Controllers\ProjectController;

Route::prefix('v1/companies/{companyId}')
    ->middleware([AuthenticateJwt::class, EnsureCompanyMember::class])
    ->group(function () {
        Route::get('issue-types', [ProjectController::class, 'issueTypes'])
            ->middleware(EnsurePermission::class.':projects.view');

        Route::get('projects', [ProjectController::class, 'index'])
            ->middleware(EnsurePermission::class.':projects.view');
        Route::post('projects', [ProjectController::class, 'store'])
            ->middleware(EnsurePermission::class.':projects.create');
        Route::get('projects/{projectId}', [ProjectController::class, 'show'])
            ->middleware(EnsurePermission::class.':projects.view');
        Route::patch('projects/{projectId}', [ProjectController::class, 'update'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::delete('projects/{projectId}', [ProjectController::class, 'destroy'])
            ->middleware(EnsurePermission::class.':projects.delete');

        Route::post('projects/{projectId}/members/{employeeId}', [ProjectController::class, 'addMember'])
            ->middleware(EnsurePermission::class.':projects.manage_members');
        Route::delete('projects/{projectId}/members/{employeeId}', [ProjectController::class, 'removeMember'])
            ->middleware(EnsurePermission::class.':projects.manage_members');
        Route::put('projects/{projectId}/lead', [ProjectController::class, 'setLead'])
            ->middleware(EnsurePermission::class.':projects.manage_members');

        Route::post('projects/{projectId}/teams/{teamId}', [ProjectController::class, 'attachTeam'])
            ->middleware(EnsurePermission::class.':projects.manage_members');
        Route::delete('projects/{projectId}/teams/{teamId}', [ProjectController::class, 'detachTeam'])
            ->middleware(EnsurePermission::class.':projects.manage_members');

        Route::get('projects/{projectId}/links', [ProjectController::class, 'links'])
            ->middleware(EnsurePermission::class.':projects.view');
        Route::post('projects/{projectId}/links', [ProjectController::class, 'createLink'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::patch('projects/{projectId}/links/{linkId}', [ProjectController::class, 'updateLink'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::delete('projects/{projectId}/links/{linkId}', [ProjectController::class, 'deleteLink'])
            ->middleware(EnsurePermission::class.':projects.update');

        Route::get('projects/{projectId}/statuses', [ProjectController::class, 'statuses'])
            ->middleware(EnsurePermission::class.':projects.view');
        Route::post('projects/{projectId}/statuses', [ProjectController::class, 'createStatus'])
            ->middleware(EnsurePermission::class.':projects.update');

        Route::get('projects/{projectId}/files', [ProjectController::class, 'files'])
            ->middleware(EnsurePermission::class.':projects.view');
        Route::post('projects/{projectId}/files', [ProjectController::class, 'attachFile'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::delete('projects/{projectId}/files/{mediaId}', [ProjectController::class, 'deleteFile'])
            ->middleware(EnsurePermission::class.':projects.update');

        Route::get('projects/{projectId}/messages', [ProjectController::class, 'messages'])
            ->middleware(EnsurePermission::class.':projects.view');
        Route::post('projects/{projectId}/messages', [ProjectController::class, 'createMessage'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::get('projects/{projectId}/messages/{messageId}', [ProjectController::class, 'showMessage'])
            ->middleware(EnsurePermission::class.':projects.view');
        Route::patch('projects/{projectId}/messages/{messageId}', [ProjectController::class, 'updateMessage'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::delete('projects/{projectId}/messages/{messageId}', [ProjectController::class, 'deleteMessage'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::post('projects/{projectId}/messages/{messageId}/comments', [ProjectController::class, 'createMessageComment'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::patch('projects/{projectId}/messages/{messageId}/comments/{commentId}', [ProjectController::class, 'updateMessageComment'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::delete('projects/{projectId}/messages/{messageId}/comments/{commentId}', [ProjectController::class, 'deleteMessageComment'])
            ->middleware(EnsurePermission::class.':projects.update');

        Route::get('projects/{projectId}/decisions', [ProjectController::class, 'decisions'])
            ->middleware(EnsurePermission::class.':projects.view');
        Route::post('projects/{projectId}/decisions', [ProjectController::class, 'createDecision'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::delete('projects/{projectId}/decisions/{decisionId}', [ProjectController::class, 'deleteDecision'])
            ->middleware(EnsurePermission::class.':projects.update');

        Route::get('projects/{projectId}/task-lists', [ProjectController::class, 'taskLists'])
            ->middleware(EnsurePermission::class.':projects.view');
        Route::post('projects/{projectId}/task-lists', [ProjectController::class, 'createTaskList'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::patch('projects/{projectId}/task-lists/{listId}', [ProjectController::class, 'updateTaskList'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::delete('projects/{projectId}/task-lists/{listId}', [ProjectController::class, 'deleteTaskList'])
            ->middleware(EnsurePermission::class.':projects.update');

        Route::post('projects/{projectId}/tasks', [ProjectController::class, 'createTask'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::get('projects/{projectId}/tasks/{taskId}', [ProjectController::class, 'showTask'])
            ->middleware(EnsurePermission::class.':projects.view');
        Route::patch('projects/{projectId}/tasks/{taskId}', [ProjectController::class, 'updateTask'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::delete('projects/{projectId}/tasks/{taskId}', [ProjectController::class, 'deleteTask'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::post('projects/{projectId}/tasks/{taskId}/toggle', [ProjectController::class, 'toggleTask'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::get('projects/{projectId}/tasks/{taskId}/time-entries', [ProjectController::class, 'taskTimeEntries'])
            ->middleware(EnsurePermission::class.':projects.view');
        Route::post('projects/{projectId}/tasks/{taskId}/comments', [ProjectController::class, 'createTaskComment'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::patch('projects/{projectId}/tasks/{taskId}/comments/{commentId}', [ProjectController::class, 'updateTaskComment'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::delete('projects/{projectId}/tasks/{taskId}/comments/{commentId}', [ProjectController::class, 'deleteTaskComment'])
            ->middleware(EnsurePermission::class.':projects.update');

        Route::get('projects/{projectId}/boards', [ProjectController::class, 'boards'])
            ->middleware(EnsurePermission::class.':projects.view');
        Route::post('projects/{projectId}/boards', [ProjectController::class, 'createBoard'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::get('projects/{projectId}/boards/{boardId}', [ProjectController::class, 'showBoard'])
            ->middleware(EnsurePermission::class.':projects.view');
        Route::patch('projects/{projectId}/boards/{boardId}', [ProjectController::class, 'updateBoard'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::delete('projects/{projectId}/boards/{boardId}', [ProjectController::class, 'deleteBoard'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::get('projects/{projectId}/boards/{boardId}/backlog', [ProjectController::class, 'backlog'])
            ->middleware(EnsurePermission::class.':projects.view');
        Route::post('projects/{projectId}/boards/{boardId}/sprints/{sprintId}/start', [ProjectController::class, 'startSprint'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::post('projects/{projectId}/boards/{boardId}/sprints/{sprintId}/toggle', [ProjectController::class, 'toggleSprint'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::post('projects/{projectId}/boards/{boardId}/sprints/{sprintId}/issues', [ProjectController::class, 'createIssue'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::post('projects/{projectId}/boards/{boardId}/sprints/{sprintId}/issues/{issueId}/order', [ProjectController::class, 'reorderIssue'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::delete('projects/{projectId}/boards/{boardId}/sprints/{sprintId}/issues/{issueId}', [ProjectController::class, 'deleteIssue'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::post('projects/{projectId}/boards/{boardId}/issues/{issueId}/assignees', [ProjectController::class, 'addIssueAssignee'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::delete('projects/{projectId}/boards/{boardId}/issues/{issueId}/assignees/{assigneeId}', [ProjectController::class, 'removeIssueAssignee'])
            ->middleware(EnsurePermission::class.':projects.update');
        Route::post('projects/{projectId}/boards/{boardId}/issues/{issueId}/points', [ProjectController::class, 'setIssuePoints'])
            ->middleware(EnsurePermission::class.':projects.update');
    });
