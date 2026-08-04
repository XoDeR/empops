<?php

namespace Modules\Time\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Time\Models\CompanyPtoPolicy;
use Modules\Time\Models\EmployeePlannedHoliday;
use Modules\Time\Services\PtoService;
use RuntimeException;

class PtoController extends Controller
{
    public function __construct(private readonly PtoService $pto) {}

    public function index(Request $request): JsonResponse
    {
        return ApiResponse::success($this->pto->listPolicies($this->company($request)));
    }

    public function store(Request $request): JsonResponse
    {
        $data = $request->validate([
            'year' => ['required', 'integer', 'min:2000', 'max:2100'],
            'default_amount_of_allowed_holidays' => ['nullable', 'numeric', 'min:0'],
            'default_amount_of_sick_days' => ['nullable', 'numeric', 'min:0'],
            'default_amount_of_pto_days' => ['nullable', 'numeric', 'min:0'],
        ]);

        return ApiResponse::success($this->pto->createPolicy($this->company($request), $data), 'PTO policy created', 201);
    }

    public function show(Request $request, string $companyId, string $policyId): JsonResponse
    {
        return ApiResponse::success($this->policy($request, $policyId));
    }

    public function update(Request $request, string $companyId, string $policyId): JsonResponse
    {
        $data = $request->validate([
            'default_amount_of_allowed_holidays' => ['sometimes', 'numeric', 'min:0'],
            'default_amount_of_sick_days' => ['sometimes', 'numeric', 'min:0'],
            'default_amount_of_pto_days' => ['sometimes', 'numeric', 'min:0'],
        ]);

        return ApiResponse::success($this->pto->updatePolicy($this->policy($request, $policyId), $data), 'PTO policy updated');
    }

    public function calendar(Request $request, string $companyId, string $policyId): JsonResponse
    {
        return ApiResponse::success($this->pto->getCalendar($this->policy($request, $policyId)));
    }

    public function toggleDay(Request $request, string $companyId, string $policyId, string $day): JsonResponse
    {
        $data = $request->validate(['is_worked' => ['required', 'boolean']]);
        try {
            $entry = $this->pto->toggleCalendarDay($this->policy($request, $policyId), $day, $data['is_worked']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($entry, 'Calendar updated');
    }

    public function balance(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        if (! $this->canViewEmployee($request, $employeeId)) {
            return ApiResponse::error('Forbidden', 403);
        }

        return ApiResponse::success($this->pto->holidayBalance($this->employee($request, $employeeId)));
    }

    public function holidays(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        if (! $this->canViewEmployee($request, $employeeId)) {
            return ApiResponse::error('Forbidden', 403);
        }

        return ApiResponse::success($this->pto->listHolidays($this->employee($request, $employeeId)));
    }

    public function storeHoliday(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        if (! $this->canManageEmployee($request, $employeeId)) {
            return ApiResponse::error('Forbidden', 403);
        }
        $data = $request->validate([
            'planned_date' => ['required', 'date'],
            'type' => ['required', 'string', 'max:50'],
            'full' => ['sometimes', 'boolean'],
            'actually_taken' => ['sometimes', 'boolean'],
        ]);

        return ApiResponse::success($this->pto->createHoliday($this->employee($request, $employeeId), $data), 'Holiday created', 201);
    }

    public function destroyHoliday(Request $request, string $companyId, string $employeeId, string $holidayId): JsonResponse
    {
        if (! $this->canManageEmployee($request, $employeeId)) {
            return ApiResponse::error('Forbidden', 403);
        }
        $employee = $this->employee($request, $employeeId);
        $holiday = EmployeePlannedHoliday::query()->where('employee_id', $employee->id)->findOrFail($holidayId);
        $this->pto->deleteHoliday($holiday);

        return ApiResponse::success(null, 'Holiday deleted');
    }

    private function company(Request $request): Company { return $request->attributes->get('company'); }

    private function policy(Request $request, string $id): CompanyPtoPolicy
    {
        return CompanyPtoPolicy::query()->where('company_id', $this->company($request)->id)->findOrFail($id);
    }

    private function employee(Request $request, string $id): Employee
    {
        return Employee::query()->where('company_id', $this->company($request)->id)->findOrFail($id);
    }

    private function canManageEmployee(Request $request, string $employeeId): bool
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        return $actor->id === $employeeId || $actor->hasPermissionTo('pto.manage');
    }

    private function canViewEmployee(Request $request, string $employeeId): bool
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        return $actor->id === $employeeId
            || $actor->hasPermissionTo('pto.manage')
            || $actor->hasAnyRole(['administrator', 'hr']);
    }
}
