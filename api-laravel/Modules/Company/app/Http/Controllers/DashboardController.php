<?php

namespace Modules\Company\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Company\Services\QuestionService;
use Modules\Employee\Models\DirectReport;
use Modules\Employee\Models\Employee;
use Modules\Employee\Services\WorklogService;
use Modules\Notification\Services\NotificationService;
use Modules\Team\Models\Team;

class DashboardController extends Controller
{
    public function __construct(
        private readonly WorklogService $worklogs,
        private readonly QuestionService $questions,
        private readonly NotificationService $notifications,
    ) {}

    public function me(Request $request): JsonResponse
    {
        return $this->shell($request, 'me', true, true);
    }

    public function team(Request $request): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $hasTeams = Team::query()
            ->where('company_id', $actor->company_id)
            ->whereHas('employees', fn ($q) => $q->where('employees.id', $actor->id))
            ->exists();

        return $this->shell($request, 'team', $hasTeams || true, false);
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

        return $this->shell($request, 'manager', true, false);
    }

    public function hr(Request $request): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        if (! $actor->hasAnyRole(['administrator', 'hr'])) {
            return ApiResponse::error('Forbidden', 403);
        }

        return $this->shell($request, 'hr', true, false);
    }

    private function shell(Request $request, string $view, bool $allowed, bool $withWidgets): JsonResponse
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

        $widgets = [];
        if ($withWidgets) {
            $today = $this->worklogs->todayForEmployee($actor);
            $active = $this->questions->activePayload($company, $actor);

            $widgets = [
                [
                    'type' => 'worklog_today',
                    'data' => [
                        'logged' => $today !== null,
                        'worklog' => $today ? $this->worklogs->payload($today) : null,
                        'consecutive_missed' => (int) $actor->consecutive_worklog_missed,
                    ],
                ],
                [
                    'type' => 'active_question',
                    'data' => $active === null ? null : [
                        'id' => $active['id'],
                        'title' => $active['title'],
                        'answered' => $active['answered'],
                    ],
                ],
                [
                    'type' => 'unread_notifications',
                    'data' => [
                        'count' => $this->notifications->unreadCount($company, $actor),
                    ],
                ],
            ];
        }

        return ApiResponse::success([
            'view' => $view,
            'widgets' => $widgets,
            'flags' => [
                'is_manager' => $isManager,
                'can_manage_hr' => $actor->hasAnyRole(['administrator', 'hr']),
                'is_admin' => $actor->hasRole('administrator'),
            ],
        ]);
    }
}
