<?php

use Illuminate\Support\Facades\Route;
use Modules\Auth\Http\Middleware\AuthenticateJwt;
use Modules\Company\Http\Middleware\EnsureCompanyMember;
use Modules\Company\Http\Middleware\EnsurePermission;
use Modules\Employee\Http\Controllers\EmployeeController;
use Modules\Employee\Http\Controllers\EmployeeStatusController;
use Modules\Employee\Http\Controllers\PositionController;

Route::prefix('v1/companies/{companyId}')
    ->middleware([AuthenticateJwt::class, EnsureCompanyMember::class])
    ->group(function () {
        Route::get('employees', [EmployeeController::class, 'index'])
            ->middleware(EnsurePermission::class.':employees.view');
        Route::post('employees', [EmployeeController::class, 'store'])
            ->middleware(EnsurePermission::class.':employees.create');
        Route::get('employees/{employeeId}', [EmployeeController::class, 'show'])
            ->middleware(EnsurePermission::class.':employees.view');
        Route::patch('employees/{employeeId}', [EmployeeController::class, 'update']);
        Route::delete('employees/{employeeId}', [EmployeeController::class, 'destroy'])
            ->middleware(EnsurePermission::class.':employees.delete');
        Route::post('employees/{employeeId}/invite', [EmployeeController::class, 'invite'])
            ->middleware(EnsurePermission::class.':employees.invite');

        Route::get('positions', [PositionController::class, 'index'])
            ->middleware(EnsurePermission::class.':positions.view');
        Route::post('positions', [PositionController::class, 'store'])
            ->middleware(EnsurePermission::class.':positions.create');
        Route::patch('positions/{positionId}', [PositionController::class, 'update'])
            ->middleware(EnsurePermission::class.':positions.update');
        Route::delete('positions/{positionId}', [PositionController::class, 'destroy'])
            ->middleware(EnsurePermission::class.':positions.delete');

        Route::get('employee-statuses', [EmployeeStatusController::class, 'index'])
            ->middleware(EnsurePermission::class.':employee-statuses.view');
        Route::post('employee-statuses', [EmployeeStatusController::class, 'store'])
            ->middleware(EnsurePermission::class.':employee-statuses.create');
        Route::patch('employee-statuses/{statusId}', [EmployeeStatusController::class, 'update'])
            ->middleware(EnsurePermission::class.':employee-statuses.update');
        Route::delete('employee-statuses/{statusId}', [EmployeeStatusController::class, 'destroy'])
            ->middleware(EnsurePermission::class.':employee-statuses.delete');
    });
