<?php

use Illuminate\Support\Facades\Route;
use Modules\Auth\Http\Middleware\AuthenticateJwt;
use Modules\Company\Http\Middleware\EnsureCompanyMember;
use Modules\Company\Http\Middleware\EnsurePermission;
use Modules\Group\Http\Controllers\GroupController;

Route::prefix('v1/companies/{companyId}')
    ->middleware([AuthenticateJwt::class, EnsureCompanyMember::class])
    ->group(function () {
        Route::get('groups', [GroupController::class, 'index'])->middleware(EnsurePermission::class.':groups.view');
        Route::post('groups', [GroupController::class, 'store'])->middleware(EnsurePermission::class.':groups.manage');
        Route::get('groups/{groupId}', [GroupController::class, 'show'])->middleware(EnsurePermission::class.':groups.view');
        Route::patch('groups/{groupId}', [GroupController::class, 'update']);
        Route::delete('groups/{groupId}', [GroupController::class, 'destroy']);
        Route::post('groups/{groupId}/members/{employeeId}', [GroupController::class, 'addMember']);
        Route::delete('groups/{groupId}/members/{employeeId}', [GroupController::class, 'removeMember']);
        Route::get('groups/{groupId}/meetings', [GroupController::class, 'listMeetings'])->middleware(EnsurePermission::class.':groups.view');
        Route::post('groups/{groupId}/meetings', [GroupController::class, 'createMeeting']);
        Route::get('groups/{groupId}/meetings/{meetingId}', [GroupController::class, 'showMeeting'])->middleware(EnsurePermission::class.':groups.view');
        Route::patch('groups/{groupId}/meetings/{meetingId}', [GroupController::class, 'updateMeeting']);
        Route::delete('groups/{groupId}/meetings/{meetingId}', [GroupController::class, 'deleteMeeting']);
        Route::post('groups/{groupId}/meetings/{meetingId}/happened', [GroupController::class, 'happened']);
        Route::put('groups/{groupId}/meetings/{meetingId}/attendance', [GroupController::class, 'attendance']);
        Route::delete('groups/{groupId}/meetings/{meetingId}/attendance/{employeeId}', [GroupController::class, 'removeAttendance']);
        Route::post('groups/{groupId}/meetings/{meetingId}/agenda', [GroupController::class, 'createAgenda']);
        Route::patch('groups/{groupId}/meetings/{meetingId}/agenda/{itemId}', [GroupController::class, 'updateAgenda']);
        Route::delete('groups/{groupId}/meetings/{meetingId}/agenda/{itemId}', [GroupController::class, 'deleteAgenda']);
        Route::post('groups/{groupId}/meetings/{meetingId}/agenda/{itemId}/decisions', [GroupController::class, 'createDecision']);
        Route::patch('groups/{groupId}/meetings/{meetingId}/agenda/{itemId}/decisions/{decisionId}', [GroupController::class, 'updateDecision']);
        Route::delete('groups/{groupId}/meetings/{meetingId}/agenda/{itemId}/decisions/{decisionId}', [GroupController::class, 'deleteDecision']);
    });
