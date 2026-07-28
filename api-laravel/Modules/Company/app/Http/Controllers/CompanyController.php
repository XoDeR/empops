<?php

namespace Modules\Company\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Company\Services\CompanyService;
use Modules\Employee\Models\Employee;
use Modules\Employee\Services\EmployeeService;
use RuntimeException;

class CompanyController extends Controller
{
    public function __construct(
        private readonly CompanyService $companies,
        private readonly EmployeeService $employees,
    ) {}

    public function index(Request $request): JsonResponse
    {
        $memberships = Employee::query()
            ->with(['company', 'roles'])
            ->where('user_id', $request->user()->id)
            ->where('locked', false)
            ->get();

        $data = $memberships->map(function (Employee $employee) {
            return [
                ...$this->companies->companyPayload($employee->company),
                'employee_id' => (string) $employee->id,
                'roles' => $employee->getRoleNames()->values()->all(),
            ];
        })->values()->all();

        return ApiResponse::success($data);
    }

    public function store(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'currency' => ['sometimes', 'string', 'size:3'],
        ]);

        $result = $this->companies->create(
            $request->user(),
            $validated['name'],
            $validated['currency'] ?? 'EUR',
        );

        return ApiResponse::success([
            'company' => $this->companies->companyPayload($result['company'], includeJoinCode: true),
            'employee' => $this->employees->employeePayload($result['employee']),
        ], 'Company created', 201);
    }

    public function join(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'code' => ['required', 'string', 'max:64'],
        ]);

        try {
            $result = $this->companies->join($request->user(), $validated['code']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success([
            'company' => $this->companies->companyPayload($result['company']),
            'employee' => $this->employees->employeePayload($result['employee']),
        ], 'Joined company');
    }

    public function show(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $employee */
        $employee = $request->attributes->get('employee');

        $includeJoinCode = $employee->hasPermissionTo('company.update');

        return ApiResponse::success([
            ...$this->companies->companyPayload($company, $includeJoinCode),
            'employee_id' => (string) $employee->id,
            'roles' => $employee->getRoleNames()->values()->all(),
        ]);
    }

    public function update(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        $validated = $request->validate([
            'name' => ['sometimes', 'string', 'max:255'],
            'currency' => ['sometimes', 'string', 'size:3'],
        ]);

        $updated = $this->companies->updateSettings($company, $validated);

        return ApiResponse::success(
            $this->companies->companyPayload($updated, includeJoinCode: true),
            'Company updated',
        );
    }
}
