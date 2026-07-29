<?php

namespace Modules\Team\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Team\Models\Ship;
use Modules\Team\Models\Team;
use Modules\Team\Services\CommunicateService;
use RuntimeException;

class ShipController extends Controller
{
    public function __construct(private readonly CommunicateService $communicate) {}

    public function index(Request $request, string $companyId, string $teamId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $team = $this->findTeam($request, $teamId);

        if (! $this->communicate->canAccessTeam($actor, $team, 'ships.view')) {
            return ApiResponse::error('Forbidden', 403);
        }

        $list = Ship::query()
            ->with('employees')
            ->where('team_id', $team->id)
            ->orderByDesc('created_at')
            ->get()
            ->map(fn (Ship $s) => $this->communicate->shipPayload($s))
            ->values()
            ->all();

        return ApiResponse::success($list);
    }

    public function store(Request $request, string $companyId, string $teamId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $team = $this->findTeam($request, $teamId);

        if (! $this->communicate->canAccessTeam($actor, $team, 'ships.create')) {
            return ApiResponse::error('Forbidden', 403);
        }

        $validated = $request->validate([
            'title' => ['required', 'string', 'max:255'],
            'description' => ['nullable', 'string'],
            'employee_ids' => ['nullable', 'array'],
            'employee_ids.*' => ['uuid'],
        ]);

        try {
            $ship = $this->communicate->createShip($team, $actor, $validated);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->communicate->shipPayload($ship), 'Ship created', 201);
    }

    public function show(Request $request, string $companyId, string $teamId, string $shipId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $team = $this->findTeam($request, $teamId);

        if (! $this->communicate->canAccessTeam($actor, $team, 'ships.view')) {
            return ApiResponse::error('Forbidden', 403);
        }

        $ship = $this->findShip($team, $shipId);

        return ApiResponse::success($this->communicate->shipPayload($ship));
    }

    public function destroy(Request $request, string $companyId, string $teamId, string $shipId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $team = $this->findTeam($request, $teamId);
        $ship = $this->findShip($team, $shipId);

        if (! $this->communicate->canManageShip($actor, $ship, 'ships.delete')) {
            return ApiResponse::error('Forbidden', 403);
        }

        $this->communicate->destroyShip($ship, $actor);

        return ApiResponse::success(null, 'Ship deleted');
    }

    private function findTeam(Request $request, string $teamId): Team
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return Team::query()
            ->where('company_id', $company->id)
            ->where('id', $teamId)
            ->firstOrFail();
    }

    private function findShip(Team $team, string $shipId): Ship
    {
        return Ship::query()
            ->with('employees')
            ->where('team_id', $team->id)
            ->where('id', $shipId)
            ->firstOrFail();
    }
}
