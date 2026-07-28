<?php

namespace Modules\Company\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Services\CompanyService;
use Modules\Employee\Services\EmployeeService;
use RuntimeException;

class InvitationController extends Controller
{
    public function __construct(
        private readonly EmployeeService $employees,
        private readonly CompanyService $companies,
    ) {}

    public function accept(Request $request, string $link): JsonResponse
    {
        try {
            $employee = $this->employees->acceptInvite($request->user(), $link);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success([
            'company' => $this->companies->companyPayload($employee->company),
            'employee' => $this->employees->employeePayload($employee),
        ], 'Invitation accepted');
    }
}
