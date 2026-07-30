<?php

use Illuminate\Support\Facades\Route;
use Modules\Auth\Http\Middleware\AuthenticateJwt;
use Modules\Company\Http\Middleware\EnsureCompanyMember;
use Modules\Company\Http\Middleware\EnsurePermission;
use Modules\Finance\Http\Controllers\FinanceController;

Route::prefix('v1/companies/{companyId}')
    ->middleware([AuthenticateJwt::class, EnsureCompanyMember::class])
    ->group(function () {
        Route::get('expense-categories', [FinanceController::class, 'categories'])
            ->middleware(EnsurePermission::class.':expenses.view');
        Route::post('expense-categories', [FinanceController::class, 'createCategory'])
            ->middleware(EnsurePermission::class.':expenses.manage_categories');
        Route::patch('expense-categories/{categoryId}', [FinanceController::class, 'updateCategory'])
            ->middleware(EnsurePermission::class.':expenses.manage_categories');
        Route::delete('expense-categories/{categoryId}', [FinanceController::class, 'deleteCategory'])
            ->middleware(EnsurePermission::class.':expenses.manage_categories');

        Route::get('expenses', [FinanceController::class, 'expenses'])
            ->middleware(EnsurePermission::class.':expenses.view');
        Route::post('expenses', [FinanceController::class, 'createExpense'])
            ->middleware(EnsurePermission::class.':expenses.view');
        Route::get('expenses/pending/manager', [FinanceController::class, 'pendingManager'])
            ->middleware(EnsurePermission::class.':expenses.view');
        Route::get('expenses/pending/accounting', [FinanceController::class, 'pendingAccounting'])
            ->middleware(EnsurePermission::class.':expenses.finalize');
        Route::get('expenses/{expenseId}', [FinanceController::class, 'showExpense'])
            ->middleware(EnsurePermission::class.':expenses.view');
        Route::delete('expenses/{expenseId}', [FinanceController::class, 'deleteExpense'])
            ->middleware(EnsurePermission::class.':expenses.view');
        Route::post('expenses/{expenseId}/manager-approve', [FinanceController::class, 'managerApprove'])
            ->middleware(EnsurePermission::class.':expenses.view');
        Route::post('expenses/{expenseId}/manager-reject', [FinanceController::class, 'managerReject'])
            ->middleware(EnsurePermission::class.':expenses.view');
        Route::post('expenses/{expenseId}/accounting-approve', [FinanceController::class, 'accountingApprove'])
            ->middleware(EnsurePermission::class.':expenses.finalize');
        Route::post('expenses/{expenseId}/accounting-reject', [FinanceController::class, 'accountingReject'])
            ->middleware(EnsurePermission::class.':expenses.finalize');

        Route::post('employees/{employeeId}/accountant', [FinanceController::class, 'grantAccountant']);
        Route::delete('employees/{employeeId}/accountant', [FinanceController::class, 'revokeAccountant']);
    });
