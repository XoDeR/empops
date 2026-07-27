<?php

namespace Modules\Core\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;

class HealthController extends Controller
{
    public function health(): JsonResponse
    {
        return ApiResponse::success(['status' => 'ok'], 'Healthy');
    }

    public function version(): JsonResponse
    {
        return ApiResponse::success([
            'name' => 'empops-laravel',
            'version' => '0.0.0',
        ], 'Version');
    }
}
