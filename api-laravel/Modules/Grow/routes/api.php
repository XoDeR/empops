<?php

use Illuminate\Support\Facades\Route;
use Modules\Auth\Http\Middleware\AuthenticateJwt;
use Modules\Company\Http\Middleware\EnsureCompanyMember;
use Modules\Company\Http\Middleware\EnsurePermission;
use Modules\Grow\Http\Controllers\GrowController;

Route::prefix('v1/companies/{companyId}')
    ->middleware([AuthenticateJwt::class, EnsureCompanyMember::class])
    ->group(function () {
        // Morale
        Route::get('morale/today', [GrowController::class, 'todayMorale']);
        Route::post('morale', [GrowController::class, 'logMorale'])
            ->middleware(EnsurePermission::class.':morale.log');
        Route::get('morale/history/company', [GrowController::class, 'companyMoraleHistory'])
            ->middleware(EnsurePermission::class.':morale.view');
        Route::get('morale/history/teams/{teamId}', [GrowController::class, 'teamMoraleHistory'])
            ->middleware(EnsurePermission::class.':morale.view');
        Route::get('employees/{employeeId}/morale', [GrowController::class, 'employeeMorale']);

        // One-on-ones
        Route::get('one-on-ones/me', [GrowController::class, 'myOneOnOnes']);
        Route::get('one-on-ones/manager', [GrowController::class, 'managerOneOnOnes']);
        Route::get('one-on-ones/{entryId}', [GrowController::class, 'showOneOnOne']);
        Route::post('one-on-ones/{entryId}/happened', [GrowController::class, 'markOneOnOneHappened']);
        Route::post('one-on-ones/{entryId}/talking-points', [GrowController::class, 'storeTalkingPoint']);
        Route::post('one-on-ones/{entryId}/talking-points/{pointId}/toggle', [GrowController::class, 'toggleTalkingPoint']);
        Route::delete('one-on-ones/{entryId}/talking-points/{pointId}', [GrowController::class, 'destroyTalkingPoint']);
        Route::post('one-on-ones/{entryId}/action-items', [GrowController::class, 'storeActionItem']);
        Route::post('one-on-ones/{entryId}/action-items/{itemId}/toggle', [GrowController::class, 'toggleActionItem']);
        Route::delete('one-on-ones/{entryId}/action-items/{itemId}', [GrowController::class, 'destroyActionItem']);
        Route::post('one-on-ones/{entryId}/notes', [GrowController::class, 'storeNote']);
        Route::delete('one-on-ones/{entryId}/notes/{noteId}', [GrowController::class, 'destroyNote']);

        // Rate your manager
        Route::get('rate-your-manager/pending', [GrowController::class, 'pendingRateAnswers']);
        Route::post('rate-your-manager/answers/{answerId}', [GrowController::class, 'submitRating']);
        Route::post('rate-your-manager/answers/{answerId}/comment', [GrowController::class, 'commentOnRating']);
        Route::get('employees/{employeeId}/rate-your-manager-surveys', [GrowController::class, 'managerSurveys']);

        // Skills
        Route::get('skills', [GrowController::class, 'listSkills'])
            ->middleware(EnsurePermission::class.':skills.view');
        Route::get('skills/search', [GrowController::class, 'searchSkills'])
            ->middleware(EnsurePermission::class.':skills.view');
        Route::get('skills/{skillId}', [GrowController::class, 'showSkill'])
            ->middleware(EnsurePermission::class.':skills.view');
        Route::patch('skills/{skillId}', [GrowController::class, 'updateSkill'])
            ->middleware(EnsurePermission::class.':skills.manage');
        Route::delete('skills/{skillId}', [GrowController::class, 'destroySkill'])
            ->middleware(EnsurePermission::class.':skills.manage');
        Route::get('employees/{employeeId}/skills', [GrowController::class, 'employeeSkills']);
        Route::post('employees/{employeeId}/skills', [GrowController::class, 'attachEmployeeSkill']);
        Route::delete('employees/{employeeId}/skills/{skillId}', [GrowController::class, 'detachEmployeeSkill']);

        // e-Coffee
        Route::get('e-coffee', [GrowController::class, 'getECoffee'])
            ->middleware(EnsurePermission::class.':e_coffee.manage');
        Route::patch('e-coffee', [GrowController::class, 'updateECoffee'])
            ->middleware(EnsurePermission::class.':e_coffee.manage');
        Route::get('e-coffee/current', [GrowController::class, 'currentECoffee']);
        Route::post('e-coffee/matches/{matchId}/happened', [GrowController::class, 'markECoffeeHappened']);
        Route::get('employees/{employeeId}/e-coffees', [GrowController::class, 'employeeECoffeeHistory']);

        // Discipline
        Route::get('discipline-cases', [GrowController::class, 'listDisciplineCases'])
            ->middleware(EnsurePermission::class.':discipline.view');
        Route::post('discipline-cases', [GrowController::class, 'storeDisciplineCase'])
            ->middleware(EnsurePermission::class.':discipline.manage');
        Route::get('discipline-cases/{caseId}', [GrowController::class, 'showDisciplineCase'])
            ->middleware(EnsurePermission::class.':discipline.view');
        Route::post('discipline-cases/{caseId}/toggle', [GrowController::class, 'toggleDisciplineCase'])
            ->middleware(EnsurePermission::class.':discipline.manage');
        Route::delete('discipline-cases/{caseId}', [GrowController::class, 'destroyDisciplineCase'])
            ->middleware(EnsurePermission::class.':discipline.manage');
        Route::post('discipline-cases/{caseId}/events', [GrowController::class, 'storeDisciplineEvent'])
            ->middleware(EnsurePermission::class.':discipline.manage');
        Route::delete('discipline-cases/{caseId}/events/{eventId}', [GrowController::class, 'destroyDisciplineEvent'])
            ->middleware(EnsurePermission::class.':discipline.manage');
        Route::post('discipline-cases/{caseId}/events/{eventId}/files', [GrowController::class, 'attachDisciplineFile'])
            ->middleware(EnsurePermission::class.':discipline.manage');
    });
