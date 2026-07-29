<?php

namespace Modules\Place\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Modules\Place\Models\Country;

class CountryController extends Controller
{
    public function index(): JsonResponse
    {
        $list = Country::query()
            ->orderBy('name')
            ->get()
            ->map(fn (Country $c) => [
                'id' => (string) $c->id,
                'name' => $c->name,
                'code' => $c->code,
            ])
            ->values()
            ->all();

        return ApiResponse::success($list);
    }
}
