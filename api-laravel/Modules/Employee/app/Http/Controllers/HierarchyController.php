<?php

namespace Modules\Employee\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\DirectReport;
use Modules\Employee\Models\Employee;
use Modules\Employee\Services\HierarchyService;
use RuntimeException;

class HierarchyController extends Controller
{
    public function __construct(private readonly HierarchyService $hierarchy) {}

    public function managers(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        $employee = $this->findEmployee($request, $employeeId);

        $list = DirectReport::query()
            ->with('manager')
            ->where('company_id', $employee->company_id)
            ->where('employee_id', $employee->id)
            ->get()
            ->map(fn (DirectReport $d) => $this->hierarchy->employeeSummary($d->manager))
            ->values()
            ->all();

        return ApiResponse::success($list);
    }

    public function directReports(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        $employee = $this->findEmployee($request, $employeeId);

        $list = DirectReport::query()
            ->with('employee')
            ->where('company_id', $employee->company_id)
            ->where('manager_id', $employee->id)
            ->get()
            ->map(fn (DirectReport $d) => $this->hierarchy->employeeSummary($d->employee))
            ->values()
            ->all();

        return ApiResponse::success($list);
    }

    public function assignManager(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $employee = $this->findEmployee($request, $employeeId);

        $validated = $request->validate([
            'manager_id' => ['required', 'uuid'],
        ]);

        $manager = Employee::query()
            ->where('company_id', $company->id)
            ->where('id', $validated['manager_id'])
            ->firstOrFail();

        try {
            $edge = $this->hierarchy->assignManager($company, $employee, $manager, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success([
            'manager' => $this->hierarchy->employeeSummary($edge->manager),
            'employee' => $this->hierarchy->employeeSummary($edge->employee),
        ], 'Manager assigned', 201);
    }

    public function unassignManager(Request $request, string $companyId, string $employeeId, string $managerId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $employee = $this->findEmployee($request, $employeeId);

        $manager = Employee::query()
            ->where('company_id', $company->id)
            ->where('id', $managerId)
            ->firstOrFail();

        try {
            $this->hierarchy->unassignManager($company, $employee, $manager, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Manager unassigned');
    }

    private function findEmployee(Request $request, string $employeeId): Employee
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return Employee::query()
            ->where('company_id', $company->id)
            ->where('id', $employeeId)
            ->firstOrFail();
    }
}
