<?php

namespace Modules\Time\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Time\Models\Timesheet;
use Modules\Time\Services\TimeService;
use RuntimeException;

class TimeController extends Controller
{
    public function __construct(private readonly TimeService $time) {}

    public function timesheet(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $data = $request->validate([
            'date' => ['nullable', 'date'],
            'employeeId' => ['nullable', 'uuid'],
            'employee_id' => ['nullable', 'uuid'],
        ]);
        $targetId = $data['employeeId'] ?? $data['employee_id'] ?? null;
        $employee = $targetId ? $this->employee($company, $targetId) : $actor;

        if ((string) $employee->id !== (string) $actor->id && ! $actor->can('timesheets.view')) {
            return ApiResponse::error('Forbidden', 403);
        }

        return ApiResponse::success(
            $this->time->payload($this->time->createOrGet($company, $employee, $data['date'] ?? null)),
        );
    }

    public function pending(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        if (! $actor->can('timesheets.approve')) {
            return ApiResponse::error('Forbidden', 403);
        }

        return ApiResponse::success(
            $this->time->pending($company, $actor)
                ->map(fn ($timesheet) => $this->time->payload($timesheet))
                ->values()
                ->all(),
        );
    }

    public function show(Request $request, string $companyId, string $timesheetId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $timesheet = $this->time->find($company, $timesheetId);

        if ((string) $timesheet->employee_id !== (string) $actor->id && ! $actor->can('timesheets.view')) {
            return ApiResponse::error('Forbidden', 403);
        }

        return ApiResponse::success($this->time->payload($timesheet));
    }

    public function upsertEntry(Request $request, string $companyId, string $timesheetId): JsonResponse
    {
        $timesheet = $this->ownedTimesheet($request, $timesheetId);
        if ($timesheet instanceof JsonResponse) {
            return $timesheet;
        }
        $data = $request->validate([
            'happened_at' => ['required', 'date'],
            'duration' => ['required', 'integer', 'min:1', 'max:1440'],
            'description' => ['nullable', 'string'],
        ]);

        try {
            return ApiResponse::success($this->time->payload($this->time->upsertEntry($timesheet, $data)));
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }
    }

    public function deleteEntry(Request $request, string $companyId, string $timesheetId, string $entryId): JsonResponse
    {
        $timesheet = $this->ownedTimesheet($request, $timesheetId);
        if ($timesheet instanceof JsonResponse) {
            return $timesheet;
        }

        try {
            return ApiResponse::success($this->time->payload($this->time->deleteEntry($timesheet, $entryId)));
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }
    }

    public function submit(Request $request, string $companyId, string $timesheetId): JsonResponse
    {
        $timesheet = $this->ownedTimesheet($request, $timesheetId);
        if ($timesheet instanceof JsonResponse) {
            return $timesheet;
        }

        try {
            return ApiResponse::success($this->time->payload($this->time->submit($timesheet)));
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }
    }

    public function approve(Request $request, string $companyId, string $timesheetId): JsonResponse
    {
        return $this->decide($request, $timesheetId, true);
    }

    public function reject(Request $request, string $companyId, string $timesheetId): JsonResponse
    {
        return $this->decide($request, $timesheetId, false);
    }

    public function setWorkFromHome(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $employee = $this->employee($company, $employeeId);
        if ((string) $employee->id !== (string) $actor->id && ! $actor->hasAnyRole(['administrator', 'hr'])) {
            return ApiResponse::error('Forbidden', 403);
        }
        $data = $request->validate([
            'date' => ['required', 'date'],
            'work_from_home' => ['required', 'boolean'],
        ]);

        try {
            $enabled = $this->time->setWorkFromHome($company, $employee, $data['date'], $data['work_from_home']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(['date' => $data['date'], 'work_from_home' => $enabled]);
    }

    public function workFromHomeSetting(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return ApiResponse::success(['enabled' => (bool) $company->work_from_home_enabled]);
    }

    public function updateWorkFromHomeSetting(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        if (! $actor->hasAnyRole(['administrator', 'hr']) && ! $actor->can('company.update')) {
            return ApiResponse::error('Forbidden', 403);
        }
        $data = $request->validate(['enabled' => ['required', 'boolean']]);
        $company->update(['work_from_home_enabled' => $data['enabled']]);

        return ApiResponse::success(['enabled' => (bool) $company->work_from_home_enabled]);
    }

    private function decide(Request $request, string $timesheetId, bool $approve): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $timesheet = $this->time->find($company, $timesheetId);
        if (! $actor->can('timesheets.approve') || ! $this->time->canApprove($actor, $timesheet)) {
            return ApiResponse::error('Forbidden', 403);
        }

        try {
            return ApiResponse::success($this->time->payload($this->time->decide($timesheet, $actor, $approve)));
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }
    }

    private function ownedTimesheet(Request $request, string $timesheetId): Timesheet|JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $timesheet = $this->time->find($company, $timesheetId);

        return (string) $timesheet->employee_id === (string) $actor->id
            ? $timesheet
            : ApiResponse::error('Forbidden', 403);
    }

    private function employee(Company $company, string $id): Employee
    {
        return Employee::query()->where('company_id', $company->id)->where('id', $id)->firstOrFail();
    }
}
