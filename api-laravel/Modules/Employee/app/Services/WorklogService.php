<?php

namespace Modules\Employee\Services;

use Carbon\Carbon;
use Illuminate\Database\QueryException;
use Illuminate\Support\Facades\DB;
use Modules\Company\Models\Company;
use Modules\Company\Services\AuditLogger;
use Modules\Employee\Models\DirectReport;
use Modules\Employee\Models\Employee;
use Modules\Employee\Models\Worklog;
use Modules\Team\Models\Team;
use RuntimeException;

final class WorklogService
{
    public function __construct(private readonly AuditLogger $audit) {}

    /**
     * @param  array{content: string, logged_on?: string|null}  $data
     */
    public function create(Company $company, Employee $actor, array $data): Worklog
    {
        $loggedOn = ! empty($data['logged_on'])
            ? Carbon::parse($data['logged_on'])->toDateString()
            : now()->toDateString();

        try {
            $worklog = Worklog::query()->create([
                'company_id' => $company->id,
                'employee_id' => $actor->id,
                'content' => trim($data['content']),
                'logged_on' => $loggedOn,
            ]);
        } catch (QueryException $e) {
            if ($this->isUniqueViolation($e)) {
                throw new RuntimeException('Worklog already logged for this day', 409);
            }
            throw $e;
        }

        $actor->consecutive_worklog_missed = 0;
        $actor->save();

        $this->audit->log($company, $actor, 'worklog.created', $worklog, [
            'logged_on' => $loggedOn,
        ]);

        return $worklog;
    }

    /**
     * @return list<array<string, mixed>>
     */
    public function listForEmployee(Employee $employee, ?string $from = null, ?string $to = null): array
    {
        $query = Worklog::query()
            ->where('employee_id', $employee->id)
            ->orderByDesc('logged_on');

        if ($from) {
            $query->whereDate('logged_on', '>=', $from);
        }
        if ($to) {
            $query->whereDate('logged_on', '<=', $to);
        }

        return $query->get()->map(fn (Worklog $w) => $this->payload($w))->values()->all();
    }

    public function destroy(Worklog $worklog, Employee $actor): void
    {
        $company = $worklog->company;
        $payload = ['logged_on' => $worklog->logged_on?->toDateString()];
        $worklog->delete();
        $this->audit->log($company, $actor, 'worklog.deleted', null, $payload);
    }

    /**
     * @return list<array<string, mixed>>
     */
    public function listForTeam(Team $team, string $date): array
    {
        $memberIds = $team->employees()->pluck('employees.id');

        return Worklog::query()
            ->with('employee')
            ->where('company_id', $team->company_id)
            ->whereDate('logged_on', $date)
            ->whereIn('employee_id', $memberIds)
            ->orderBy('logged_on')
            ->get()
            ->map(fn (Worklog $w) => array_merge($this->payload($w), [
                'employee' => $w->employee ? [
                    'id' => (string) $w->employee->id,
                    'first_name' => $w->employee->first_name,
                    'last_name' => $w->employee->last_name,
                    'email' => $w->employee->email,
                ] : null,
            ]))
            ->values()
            ->all();
    }

    public function canView(Employee $actor, Employee $subject): bool
    {
        if ((string) $actor->id === (string) $subject->id) {
            return true;
        }
        if ($actor->hasPermissionTo('worklogs.view')) {
            return true;
        }

        return DirectReport::query()
            ->where('company_id', $actor->company_id)
            ->where('manager_id', $actor->id)
            ->where('employee_id', $subject->id)
            ->exists();
    }

    public function canDelete(Employee $actor, Employee $subject): bool
    {
        if ((string) $actor->id === (string) $subject->id) {
            return true;
        }
        if ($actor->hasPermissionTo('worklogs.delete')) {
            return true;
        }

        return DirectReport::query()
            ->where('company_id', $actor->company_id)
            ->where('manager_id', $actor->id)
            ->where('employee_id', $subject->id)
            ->exists();
    }

    public function isTeamMemberOrCanView(Employee $actor, Team $team): bool
    {
        if ($actor->hasPermissionTo('worklogs.view')) {
            return true;
        }

        return $team->employees()->where('employees.id', $actor->id)->exists();
    }

    public function markMissedForDate(string $date): int
    {
        return DB::update("
            UPDATE employees
            SET consecutive_worklog_missed = consecutive_worklog_missed + 1,
                updated_at = NOW()
            WHERE locked = false
              AND NOT EXISTS (
                SELECT 1 FROM worklogs
                WHERE worklogs.employee_id = employees.id
                  AND worklogs.logged_on = ?
              )
        ", [$date]);
    }

    /**
     * @return array<string, mixed>
     */
    public function payload(Worklog $worklog): array
    {
        return [
            'id' => (string) $worklog->id,
            'company_id' => (string) $worklog->company_id,
            'employee_id' => (string) $worklog->employee_id,
            'content' => $worklog->content,
            'logged_on' => $worklog->logged_on?->toDateString(),
            'created_at' => $worklog->created_at?->toIso8601String(),
            'updated_at' => $worklog->updated_at?->toIso8601String(),
        ];
    }

    public function todayForEmployee(Employee $employee): ?Worklog
    {
        return Worklog::query()
            ->where('employee_id', $employee->id)
            ->whereDate('logged_on', now()->toDateString())
            ->first();
    }

    private function isUniqueViolation(QueryException $e): bool
    {
        return ($e->errorInfo[0] ?? null) === '23505'
            || str_contains($e->getMessage(), 'UNIQUE')
            || str_contains($e->getMessage(), 'unique');
    }
}
