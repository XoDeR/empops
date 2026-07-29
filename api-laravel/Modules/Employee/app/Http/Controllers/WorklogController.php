<?php

namespace Modules\Employee\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Employee\Models\Worklog;
use Modules\Employee\Services\WorklogService;
use Modules\Team\Models\Team;
use RuntimeException;

class WorklogController extends Controller
{
    public function __construct(private readonly WorklogService $worklogs) {}

    public function store(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $validated = $request->validate([
            'content' => ['required', 'string'],
            'logged_on' => ['nullable', 'date_format:Y-m-d'],
        ]);

        try {
            $worklog = $this->worklogs->create($company, $actor, $validated);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->worklogs->payload($worklog), 'Worklog created', 201);
    }

    public function indexForEmployee(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $employee = $this->findEmployee($request, $employeeId);

        if (! $this->worklogs->canView($actor, $employee)) {
            return ApiResponse::error('Forbidden', 403);
        }

        $from = $request->query('from');
        $to = $request->query('to');

        return ApiResponse::success($this->worklogs->listForEmployee(
            $employee,
            is_string($from) ? $from : null,
            is_string($to) ? $to : null,
        ));
    }

    public function destroy(Request $request, string $companyId, string $employeeId, string $worklogId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $employee = $this->findEmployee($request, $employeeId);

        if (! $this->worklogs->canDelete($actor, $employee)) {
            return ApiResponse::error('Forbidden', 403);
        }

        $worklog = Worklog::query()
            ->where('company_id', $employee->company_id)
            ->where('employee_id', $employee->id)
            ->where('id', $worklogId)
            ->firstOrFail();

        $this->worklogs->destroy($worklog, $actor);

        return ApiResponse::success(null, 'Worklog deleted');
    }

    public function indexForTeam(Request $request, string $companyId, string $teamId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $team = Team::query()
            ->where('company_id', $company->id)
            ->where('id', $teamId)
            ->firstOrFail();

        if (! $this->worklogs->isTeamMemberOrCanView($actor, $team)) {
            return ApiResponse::error('Forbidden', 403);
        }

        $date = $request->query('date');
        $date = is_string($date) && $date !== '' ? $date : now()->toDateString();

        return ApiResponse::success($this->worklogs->listForTeam($team, $date));
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
