<?php

use Illuminate\Support\Facades\Route;
use Modules\Auth\Http\Middleware\AuthenticateJwt;
use Modules\Company\Http\Middleware\EnsureCompanyMember;
use Modules\Company\Http\Middleware\EnsurePermission;
use Modules\Time\Http\Controllers\TimeController;

Route::prefix('v1/companies/{companyId}')
    ->middleware([AuthenticateJwt::class, EnsureCompanyMember::class])
    ->group(function () {
        Route::get('timesheets', [TimeController::class, 'timesheet'])
            ->middleware(EnsurePermission::class.':timesheets.view');
        Route::post('timesheets', [TimeController::class, 'timesheet'])
            ->middleware(EnsurePermission::class.':timesheets.view');
        Route::get('timesheets/pending', [TimeController::class, 'pending'])
            ->middleware(EnsurePermission::class.':timesheets.approve');
        Route::get('timesheets/{timesheetId}', [TimeController::class, 'show'])
            ->middleware(EnsurePermission::class.':timesheets.view');
        Route::post('timesheets/{timesheetId}/entries', [TimeController::class, 'upsertEntry'])
            ->middleware(EnsurePermission::class.':timesheets.view');
        Route::delete('timesheets/{timesheetId}/entries/{entryId}', [TimeController::class, 'deleteEntry'])
            ->middleware(EnsurePermission::class.':timesheets.view');
        Route::post('timesheets/{timesheetId}/submit', [TimeController::class, 'submit'])
            ->middleware(EnsurePermission::class.':timesheets.view');
        Route::post('timesheets/{timesheetId}/approve', [TimeController::class, 'approve'])
            ->middleware(EnsurePermission::class.':timesheets.approve');
        Route::post('timesheets/{timesheetId}/reject', [TimeController::class, 'reject'])
            ->middleware(EnsurePermission::class.':timesheets.approve');

        Route::put('employees/{employeeId}/work-from-home', [TimeController::class, 'setWorkFromHome']);
        Route::get('work-from-home', [TimeController::class, 'workFromHomeSetting']);
        Route::patch('work-from-home', [TimeController::class, 'updateWorkFromHomeSetting']);
    });
