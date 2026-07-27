<?php

use Illuminate\Support\Facades\Route;
use Modules\Core\Http\Controllers\HealthController;

Route::prefix('v1')->group(function () {
    Route::get('health', [HealthController::class, 'health']);
    Route::get('version', [HealthController::class, 'version']);
});
