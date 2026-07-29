<?php

use Illuminate\Support\Facades\Route;
use Modules\Auth\Http\Middleware\AuthenticateJwt;
use Modules\Company\Http\Middleware\EnsureCompanyMember;
use Modules\Place\Http\Controllers\CountryController;
use Modules\Place\Http\Controllers\PlaceController;

Route::prefix('v1')->middleware(AuthenticateJwt::class)->group(function () {
    Route::get('countries', [CountryController::class, 'index']);

    Route::prefix('companies/{companyId}')
        ->middleware(EnsureCompanyMember::class)
        ->group(function () {
            Route::get('employees/{employeeId}/places', [PlaceController::class, 'index']);
            Route::post('employees/{employeeId}/places', [PlaceController::class, 'store']);
            Route::patch('places/{placeId}', [PlaceController::class, 'update']);
            Route::put('places/{placeId}/activate', [PlaceController::class, 'activate']);
            Route::delete('places/{placeId}', [PlaceController::class, 'destroy']);
        });
});
