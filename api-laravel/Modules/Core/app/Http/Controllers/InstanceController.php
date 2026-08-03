<?php

namespace Modules\Core\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;

class InstanceController extends Controller
{
    public function show(): JsonResponse
    {
        return ApiResponse::success([
            'enable_signups' => (bool) config('empops.enable_signups'),
            'demo_mode' => (bool) config('empops.demo_mode'),
            'enable_paid_plan' => (bool) config('empops.enable_paid_plan'),
        ]);
    }
}
