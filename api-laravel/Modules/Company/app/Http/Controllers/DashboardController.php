<?php

namespace Modules\Company\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Company\Services\AuditLogger;
use Modules\Employee\Models\Employee;
use Modules\Employee\Models\DirectReport;
use Modules\Team\Models\Team;

class DashboardController extends Controller
{
    public function me(Request $request): JsonResponse
    {
        return $this->shell($request, 'me', true);
    }

    public function team(Request $request): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $hasTeams = Team::query()
            ->where('company_id', $actor->company_id)
            ->whereHas('employees', fn ($q) => $q->where('employees.id', $actor->id))
            ->exists();

        return $this->shell($request, 'team', $hasTeams || true);
    }

    public function manager(Request $request): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $isManager = $actor->hasRole('manager')
            || DirectReport::query()
                ->where('company_id', $actor->company_id)
                ->where('manager_id', $actor->id)
                ->exists();

        if (! $isManager) {
            return ApiResponse::error('Forbidden', 403);
        }

        return $this->shell($request, 'manager', true);
    }

    public function hr(Request $request): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        if (! $actor->hasAnyRole(['administrator', 'hr'])) {
            return ApiResponse::error('Forbidden', 403);
        }

        return $this->shell($request, 'hr', true);
    }

    private function shell(Request $request, string $view, bool $allowed): JsonResponse
    {
        if (! $allowed) {
            return ApiResponse::error('Forbidden', 403);
        }

        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $isManager = $actor->hasRole('manager')
            || DirectReport::query()
                ->where('company_id', $company->id)
                ->where('manager_id', $actor->id)
                ->exists();

        return ApiResponse::success([
            'view' => $view,
            'widgets' => [],
            'flags' => [
                'is_manager' => $isManager,
                'can_manage_hr' => $actor->hasAnyRole(['administrator', 'hr']),
                'is_admin' => $actor->hasRole('administrator'),
            ],
        ]);
    }
}
