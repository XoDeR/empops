<?php

namespace Modules\Billing\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Billing\Services\BillingService;
use Modules\Company\Models\Company;

class BillingController extends Controller
{
    public function __construct(private readonly BillingService $billing) {}

    public function invoices(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return ApiResponse::success($this->billing->listInvoices($company));
    }
}
