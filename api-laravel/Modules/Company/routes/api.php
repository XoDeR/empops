<?php

use Illuminate\Support\Facades\Route;
use Modules\Auth\Http\Middleware\AuthenticateJwt;
use Modules\Company\Http\Controllers\AuditLogController;
use Modules\Company\Http\Controllers\CompanyController;
use Modules\Company\Http\Controllers\CompanyNewsController;
use Modules\Company\Http\Controllers\DashboardController;
use Modules\Company\Http\Controllers\InvitationController;
use Modules\Company\Http\Controllers\QuestionController;
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

            Route::get('news', [CompanyNewsController::class, 'index']);
            Route::get('news/{newsId}', [CompanyNewsController::class, 'show']);
            Route::post('news', [CompanyNewsController::class, 'store'])
                ->middleware(EnsurePermission::class.':news.create');
            Route::patch('news/{newsId}', [CompanyNewsController::class, 'update'])
                ->middleware(EnsurePermission::class.':news.update');
            Route::delete('news/{newsId}', [CompanyNewsController::class, 'destroy'])
                ->middleware(EnsurePermission::class.':news.delete');

            Route::get('questions', [QuestionController::class, 'index']);
            Route::get('questions/active', [QuestionController::class, 'active']);
            Route::get('questions/{questionId}', [QuestionController::class, 'show']);
            Route::post('questions', [QuestionController::class, 'store'])
                ->middleware(EnsurePermission::class.':questions.create');
            Route::patch('questions/{questionId}', [QuestionController::class, 'update'])
                ->middleware(EnsurePermission::class.':questions.update');
            Route::delete('questions/{questionId}', [QuestionController::class, 'destroy'])
                ->middleware(EnsurePermission::class.':questions.delete');
            Route::put('questions/{questionId}/activate', [QuestionController::class, 'activate'])
                ->middleware(EnsurePermission::class.':questions.manage');
            Route::put('questions/{questionId}/deactivate', [QuestionController::class, 'deactivate'])
                ->middleware(EnsurePermission::class.':questions.manage');
            Route::post('questions/{questionId}/answers', [QuestionController::class, 'storeAnswer']);
            Route::patch('questions/{questionId}/answers/{answerId}', [QuestionController::class, 'updateAnswer']);
            Route::delete('questions/{questionId}/answers/{answerId}', [QuestionController::class, 'destroyAnswer']);

            Route::get('audit-logs', [AuditLogController::class, 'index'])
                ->middleware(EnsurePermission::class.':adminland.access');
        });
});
