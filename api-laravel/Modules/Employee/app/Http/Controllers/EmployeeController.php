<?php

namespace Modules\Employee\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Employee\Services\EmployeeService;
use RuntimeException;

class EmployeeController extends Controller
{
    public function __construct(private readonly EmployeeService $employees) {}

    public function index(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        $list = Employee::query()
            ->with(['position', 'status', 'roles'])
            ->where('company_id', $company->id)
            ->orderBy('last_name')
            ->orderBy('first_name')
            ->get()
            ->map(fn (Employee $e) => $this->employees->employeePayload($e))
            ->values()
            ->all();

        return ApiResponse::success($list);
    }

    public function store(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        $validated = $request->validate([
            'email' => ['required', 'email', 'max:255'],
            'first_name' => ['required', 'string', 'max:255'],
            'last_name' => ['required', 'string', 'max:255'],
            'hired_at' => ['nullable', 'date'],
            'position_id' => ['nullable', 'uuid'],
            'employee_status_id' => ['nullable', 'uuid'],
            'role' => ['sometimes', 'in:administrator,hr,employee'],
        ]);

        $employee = $this->employees->create(
            $company,
            $validated,
            $validated['role'] ?? 'employee',
        );

        return ApiResponse::success(
            $this->employees->employeePayload($employee),
            'Employee created',
            201,
        );
    }

    public function show(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        $employee = $this->findInCompany($request, $employeeId);

        return ApiResponse::success($this->employees->employeePayload($employee));
    }

    public function update(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $employee = $this->findInCompany($request, $employeeId);

        $canManage = $actor->hasPermissionTo('employees.update');
        $isSelf = (string) $actor->id === (string) $employee->id;

        if (! $canManage && ! $isSelf) {
            return ApiResponse::error('Forbidden', 403);
        }

        $rules = [
            'first_name' => ['sometimes', 'string', 'max:255'],
            'last_name' => ['sometimes', 'string', 'max:255'],
        ];

        if ($canManage) {
            $rules = array_merge($rules, [
                'email' => ['sometimes', 'email', 'max:255'],
                'hired_at' => ['nullable', 'date'],
                'position_id' => ['nullable', 'uuid'],
                'employee_status_id' => ['nullable', 'uuid'],
                'locked' => ['sometimes', 'boolean'],
                'role' => ['sometimes', 'in:administrator,hr,employee'],
            ]);
        }

        $validated = $request->validate($rules);
        $updated = $this->employees->update($employee, $validated);

        return ApiResponse::success($this->employees->employeePayload($updated), 'Employee updated');
    }

    public function destroy(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        $employee = $this->findInCompany($request, $employeeId);
        $employee->delete();

        return ApiResponse::success(null, 'Employee deleted');
    }

    public function invite(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        $employee = $this->findInCompany($request, $employeeId);

        try {
            $invited = $this->employees->invite($employee);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(
            $this->employees->employeePayload($invited, includeInvite: true),
            'Invitation created',
        );
    }

    public function import(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        $request->validate([
            'file' => ['required', 'file', 'mimes:csv,txt', 'max:5120'],
        ]);

        $file = $request->file('file');
        if ($file === null) {
            return ApiResponse::error('File required', 422);
        }

        try {
            $result = $this->employees->importFromCsv($company, $file->getRealPath());
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($result, 'Import finished');
    }

    private function findInCompany(Request $request, string $employeeId): Employee
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return Employee::query()
            ->with(['position', 'status', 'roles'])
            ->where('company_id', $company->id)
            ->where('id', $employeeId)
            ->firstOrFail();
    }
}
