<?php

namespace Modules\Team\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Team\Models\Team;
use Modules\Team\Services\TeamService;
use RuntimeException;

class TeamController extends Controller
{
    public function __construct(private readonly TeamService $teams) {}

    public function index(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        $list = Team::query()
            ->with(['leader', 'employees'])
            ->where('company_id', $company->id)
            ->orderBy('name')
            ->get()
            ->map(fn (Team $t) => $this->teams->teamPayload($t))
            ->values()
            ->all();

        return ApiResponse::success($list);
    }

    public function store(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $validated = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'description' => ['nullable', 'string'],
        ]);

        try {
            $team = $this->teams->create($company, $validated, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->teams->teamPayload($team), 'Team created', 201);
    }

    public function show(Request $request, string $companyId, string $teamId): JsonResponse
    {
        $team = $this->findInCompany($request, $teamId);

        return ApiResponse::success($this->teams->teamPayload($team));
    }

    public function update(Request $request, string $companyId, string $teamId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $team = $this->findInCompany($request, $teamId);

        $validated = $request->validate([
            'name' => ['sometimes', 'string', 'max:255'],
            'description' => ['nullable', 'string'],
        ]);

        try {
            $updated = $this->teams->update($team, $validated, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->teams->teamPayload($updated), 'Team updated');
    }

    public function destroy(Request $request, string $companyId, string $teamId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $team = $this->findInCompany($request, $teamId);
        $this->teams->destroy($team, $actor);

        return ApiResponse::success(null, 'Team deleted');
    }

    public function addMember(Request $request, string $companyId, string $teamId, string $employeeId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $team = $this->findInCompany($request, $teamId);
        $employee = $this->findEmployee($company, $employeeId);

        try {
            $updated = $this->teams->addMember($team, $employee, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->teams->teamPayload($updated), 'Member added');
    }

    public function removeMember(Request $request, string $companyId, string $teamId, string $employeeId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $team = $this->findInCompany($request, $teamId);
        $employee = $this->findEmployee($company, $employeeId);

        try {
            $updated = $this->teams->removeMember($team, $employee, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->teams->teamPayload($updated), 'Member removed');
    }

    public function setLead(Request $request, string $companyId, string $teamId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $team = $this->findInCompany($request, $teamId);

        $validated = $request->validate([
            'employee_id' => ['nullable', 'uuid'],
        ]);

        $leader = null;
        if (! empty($validated['employee_id'])) {
            $leader = $this->findEmployee($company, $validated['employee_id']);
        }

        try {
            $updated = $this->teams->setLead($team, $leader, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->teams->teamPayload($updated), 'Team lead updated');
    }

    private function findInCompany(Request $request, string $teamId): Team
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return Team::query()
            ->with(['leader', 'employees'])
            ->where('company_id', $company->id)
            ->where('id', $teamId)
            ->firstOrFail();
    }

    private function findEmployee(Company $company, string $employeeId): Employee
    {
        return Employee::query()
            ->where('company_id', $company->id)
            ->where('id', $employeeId)
            ->firstOrFail();
    }
}
