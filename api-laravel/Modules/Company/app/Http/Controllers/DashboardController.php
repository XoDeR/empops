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
use Modules\Finance\Services\FinanceService;
use Modules\Grow\Services\GrowService;
use Modules\Notification\Services\NotificationService;
use Modules\Team\Models\Team;
use Modules\Time\Services\TimeService;

class DashboardController extends Controller
{
    public function __construct(
        private readonly WorklogService $worklogs,
        private readonly QuestionService $questions,
        private readonly NotificationService $notifications,
        private readonly TimeService $time,
        private readonly FinanceService $finance,
        private readonly GrowService $grow,
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

    public function accountant(Request $request): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        return $this->shell($request, 'accountant', $actor->hasRole('accountant'), false);
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
                [
                    'type' => 'timesheet_current_week',
                    'data' => $this->time->currentWeekPayload($company, $actor),
                ],
            ];

            $todayMorale = $this->grow->todayMorale($actor);
            $widgets[] = [
                'type' => 'morale_today',
                'data' => [
                    'logged' => $todayMorale !== null,
                    'morale' => $todayMorale ? $this->grow->moralePayload($todayMorale) : null,
                ],
            ];

            $this->grow->ensureOpenOneOnOnesForEmployee($company, $actor);
            $openOnes = $this->grow->listOpenOneOnOnesForEmployee($company, $actor);
            $widgets[] = [
                'type' => 'one_on_one_current',
                'data' => [
                    'entries' => $openOnes,
                ],
            ];

            $pendingRates = $this->grow->pendingRateAnswers($actor);
            if (count($pendingRates) > 0) {
                $widgets[] = [
                    'type' => 'rate_manager_pending',
                    'data' => [
                        'answers' => $pendingRates,
                    ],
                ];
            }

            if ($company->e_coffee_enabled) {
                $widgets[] = [
                    'type' => 'e_coffee_current',
                    'data' => [
                        'match' => $this->grow->currentECoffeeMatch($company, $actor),
                    ],
                ];
            }

            if ($company->work_from_home_enabled) {
                $widgets[] = [
                    'type' => 'wfh_today',
                    'data' => [
                        'work_from_home' => $this->time->isWorkFromHomeToday($actor),
                    ],
                ];
            }
        } elseif ($view === 'manager') {
            $this->grow->ensureOpenOneOnOnesForManager($company, $actor);
            $openOnes = $this->grow->listOpenOneOnOnesForManager($company, $actor);
            $widgets = [
                [
                    'type' => 'pending_timesheets',
                    'data' => ['count' => $this->time->pending($company, $actor)->count()],
                ],
                [
                    'type' => 'pending_expenses',
                    'data' => ['count' => $this->finance->pendingManager($company, $actor)->count()],
                ],
                [
                    'type' => 'one_on_ones_open',
                    'data' => [
                        'count' => count($openOnes),
                        'entries' => $openOnes,
                    ],
                ],
                [
                    'type' => 'discipline_active',
                    'data' => ['count' => $this->grow->activeDisciplineCount($company, $actor)],
                ],
            ];
        } elseif ($view === 'hr') {
            $widgets = [
                [
                    'type' => 'pending_timesheets',
                    'data' => ['count' => $this->time->pending($company, $actor)->count()],
                ],
                [
                    'type' => 'discipline_active',
                    'data' => ['count' => $this->grow->activeDisciplineCount($company, $actor)],
                ],
            ];
        } elseif ($view === 'accountant') {
            $widgets = [
                [
                    'type' => 'pending_accounting_expenses',
                    'data' => ['count' => $this->finance->pendingAccounting($company)->count()],
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
                'is_accountant' => $actor->hasRole('accountant'),
            ],
        ]);
    }
}
