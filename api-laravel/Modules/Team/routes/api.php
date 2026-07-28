<?php

use Illuminate\Support\Facades\Route;
use Modules\Auth\Http\Middleware\AuthenticateJwt;
use Modules\Company\Http\Middleware\EnsureCompanyMember;
use Modules\Company\Http\Middleware\EnsurePermission;
use Modules\Team\Http\Controllers\TeamController;

Route::prefix('v1/companies/{companyId}')
    ->middleware([AuthenticateJwt::class, EnsureCompanyMember::class])
    ->group(function () {
        Route::get('teams', [TeamController::class, 'index'])
            ->middleware(EnsurePermission::class.':teams.view');
        Route::post('teams', [TeamController::class, 'store'])
            ->middleware(EnsurePermission::class.':teams.create');
        Route::get('teams/{teamId}', [TeamController::class, 'show'])
            ->middleware(EnsurePermission::class.':teams.view');
        Route::patch('teams/{teamId}', [TeamController::class, 'update'])
            ->middleware(EnsurePermission::class.':teams.update');
        Route::delete('teams/{teamId}', [TeamController::class, 'destroy'])
            ->middleware(EnsurePermission::class.':teams.delete');

        Route::post('teams/{teamId}/members/{employeeId}', [TeamController::class, 'addMember'])
            ->middleware(EnsurePermission::class.':teams.manage_members');
        Route::delete('teams/{teamId}/members/{employeeId}', [TeamController::class, 'removeMember'])
            ->middleware(EnsurePermission::class.':teams.manage_members');
        Route::put('teams/{teamId}/lead', [TeamController::class, 'setLead'])
            ->middleware(EnsurePermission::class.':teams.manage_members');
    });
