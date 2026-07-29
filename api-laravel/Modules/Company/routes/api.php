<?php

use Illuminate\Support\Facades\Route;
use Modules\Auth\Http\Middleware\AuthenticateJwt;
use Modules\Company\Http\Controllers\AuditLogController;
use Modules\Company\Http\Controllers\CompanyController;
use Modules\Company\Http\Controllers\DashboardController;
use Modules\Company\Http\Controllers\InvitationController;
use Modules\Company\Http\Middleware\EnsureCompanyMember;
use Modules\Company\Http\Middleware\EnsurePermission;

Route::prefix('v1')->middleware(AuthenticateJwt::class)->group(function () {
    Route::get('companies', [CompanyController::class, 'index']);
    Route::post('companies', [CompanyController::class, 'store']);
    Route::post('companies/join', [CompanyController::class, 'join']);

    Route::post('invitations/{link}/accept', [InvitationController::class, 'accept']);

    Route::prefix('companies/{companyId}')
        ->middleware(EnsureCompanyMember::class)
        ->group(function () {
            Route::get('/', [CompanyController::class, 'show']);
            Route::patch('/', [CompanyController::class, 'update'])
                ->middleware(EnsurePermission::class.':company.update');
            Route::put('logo', [\Modules\Company\Http\Controllers\CompanyLogoController::class, 'update'])
                ->middleware(EnsurePermission::class.':company.update');

            Route::get('dashboard/me', [DashboardController::class, 'me']);
            Route::get('dashboard/team', [DashboardController::class, 'team']);
            Route::get('dashboard/manager', [DashboardController::class, 'manager']);
            Route::get('dashboard/hr', [DashboardController::class, 'hr']);

            Route::get('audit-logs', [AuditLogController::class, 'index'])
                ->middleware(EnsurePermission::class.':adminland.access');
        });
});
