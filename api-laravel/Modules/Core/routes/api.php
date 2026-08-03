<?php

use Illuminate\Support\Facades\Route;
use Modules\Core\Http\Controllers\HealthController;
use Modules\Core\Http\Controllers\InstanceController;

Route::prefix('v1')->group(function () {
    Route::get('health', [HealthController::class, 'health']);
    Route::get('version', [HealthController::class, 'version']);
    Route::get('instance', [InstanceController::class, 'show']);
});
