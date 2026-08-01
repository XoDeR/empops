<?php

use Illuminate\Support\Facades\Route;
use Modules\Auth\Http\Middleware\AuthenticateJwt;
use Modules\Company\Http\Middleware\EnsureCompanyMember;
use Modules\Company\Http\Middleware\EnsurePermission;
use Modules\Hardware\Http\Controllers\HardwareController;

Route::prefix('v1/companies/{companyId}')
    ->middleware([AuthenticateJwt::class, EnsureCompanyMember::class])
    ->group(function () {
        Route::get('hardware', [HardwareController::class, 'listHardware'])
            ->middleware(EnsurePermission::class.':hardware.view');
        Route::post('hardware', [HardwareController::class, 'storeHardware'])
            ->middleware(EnsurePermission::class.':hardware.manage');
        Route::get('hardware/{hardwareId}', [HardwareController::class, 'showHardware'])
            ->middleware(EnsurePermission::class.':hardware.view');
        Route::patch('hardware/{hardwareId}', [HardwareController::class, 'updateHardware'])
            ->middleware(EnsurePermission::class.':hardware.manage');
        Route::delete('hardware/{hardwareId}', [HardwareController::class, 'destroyHardware'])
            ->middleware(EnsurePermission::class.':hardware.manage');
        Route::post('hardware/{hardwareId}/lend', [HardwareController::class, 'lendHardware'])
            ->middleware(EnsurePermission::class.':hardware.manage');
        Route::post('hardware/{hardwareId}/regain', [HardwareController::class, 'regainHardware'])
            ->middleware(EnsurePermission::class.':hardware.manage');
        Route::get('employees/{employeeId}/hardware', [HardwareController::class, 'employeeHardware']);

        Route::get('softwares', [HardwareController::class, 'listSoftwares'])
            ->middleware(EnsurePermission::class.':software.view');
        Route::post('softwares', [HardwareController::class, 'storeSoftware'])
            ->middleware(EnsurePermission::class.':software.manage');
        Route::get('softwares/{softwareId}', [HardwareController::class, 'showSoftware'])
            ->middleware(EnsurePermission::class.':software.view');
        Route::patch('softwares/{softwareId}', [HardwareController::class, 'updateSoftware'])
            ->middleware(EnsurePermission::class.':software.manage');
        Route::delete('softwares/{softwareId}', [HardwareController::class, 'destroySoftware'])
            ->middleware(EnsurePermission::class.':software.manage');
        Route::post('softwares/{softwareId}/seats', [HardwareController::class, 'giveSeat'])
            ->middleware(EnsurePermission::class.':software.manage');
        Route::post('softwares/{softwareId}/seats/all', [HardwareController::class, 'giveSeatsToAll'])
            ->middleware(EnsurePermission::class.':software.manage');
        Route::delete('softwares/{softwareId}/seats/{employeeId}', [HardwareController::class, 'revokeSeat'])
            ->middleware(EnsurePermission::class.':software.manage');
        Route::get('softwares/{softwareId}/employees-without', [HardwareController::class, 'employeesWithout'])
            ->middleware(EnsurePermission::class.':software.view');
        Route::post('softwares/{softwareId}/files', [HardwareController::class, 'attachFile'])
            ->middleware(EnsurePermission::class.':software.manage');
        Route::delete('softwares/{softwareId}/files/{mediaId}', [HardwareController::class, 'detachFile'])
            ->middleware(EnsurePermission::class.':software.manage');
        Route::get('employees/{employeeId}/softwares', [HardwareController::class, 'employeeSoftwares']);
    });
