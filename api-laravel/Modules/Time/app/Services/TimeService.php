<?php

namespace Modules\Time\Services;

use Carbon\CarbonImmutable;
use Illuminate\Support\Collection;
use Modules\Company\Models\Company;
use Modules\Employee\Models\DirectReport;
use Modules\Employee\Models\Employee;
use Modules\Project\Models\Project;
use Modules\Project\Models\ProjectTask;
use Modules\Project\Services\ProjectService;
use Modules\Time\Models\EmployeeWorkFromHome;
use Modules\Time\Models\Timesheet;
use Modules\Time\Models\TimeTrackingEntry;
use RuntimeException;

final class TimeService
{
    public function __construct(
        private readonly ProjectService $projects,
    ) {}
    public function createOrGet(Company $company, Employee $employee, ?string $date = null): Timesheet
    {
        $start = CarbonImmutable::parse($date ?? now()->toDateString())->startOfWeek();

        return Timesheet::query()->firstOrCreate(
            ['employee_id' => $employee->id, 'started_at' => $start->toDateString()],
            [
                'company_id' => $company->id,
                'ended_at' => $start->endOfWeek()->toDateString(),
                'status' => 'open',
            ],
        )->load(['employee', 'approver', 'entries']);
    }

    public function find(Company $company, string $id): Timesheet
    {
        return Timesheet::query()
            ->with(['employee', 'approver', 'entries'])
            ->where('company_id', $company->id)
            ->where('id', $id)
            ->firstOrFail();
    }

    public function upsertEntry(Timesheet $timesheet, array $data): Timesheet
    {
        if (! in_array($timesheet->status, ['open', 'rejected'], true)) {
            throw new RuntimeException('Only open or rejected timesheets can be edited', 409);
        }

        $date = CarbonImmutable::parse($data['happened_at']);
        if ($date->lt($timesheet->started_at) || $date->gt($timesheet->ended_at)) {
            throw new RuntimeException('Entry date must be within the timesheet week', 422);
        }

        $projectId = $data['project_id'] ?? null;
        $projectTaskId = $data['project_task_id'] ?? null;

        if ($projectTaskId !== null) {
            $task = ProjectTask::query()
                ->with('project')
                ->where('id', $projectTaskId)
                ->firstOrFail();

            if ($projectId !== null && (string) $task->project_id !== (string) $projectId) {
                throw new RuntimeException('Task does not belong to the specified project', 422);
            }

            if ((string) $task->project->company_id !== (string) $timesheet->company_id) {
                throw new RuntimeException('Project does not belong to this company', 422);
            }

            $projectId = (string) $task->project_id;

            $match = [
                'timesheet_id' => $timesheet->id,
                'happened_at' => $date->toDateString(),
                'project_task_id' => $projectTaskId,
            ];
        } else {
            if ($projectId !== null) {
                Project::query()
                    ->where('company_id', $timesheet->company_id)
                    ->where('id', $projectId)
                    ->firstOrFail();
            }

            $match = [
                'timesheet_id' => $timesheet->id,
                'happened_at' => $date->toDateString(),
                'project_id' => null,
                'project_task_id' => null,
            ];
        }

        $existing = TimeTrackingEntry::query()
            ->where('timesheet_id', $timesheet->id)
            ->whereDate('happened_at', $date->toDateString())
            ->when(
                $projectTaskId !== null,
                fn ($query) => $query->where('project_task_id', $projectTaskId),
                fn ($query) => $query->whereNull('project_id')->whereNull('project_task_id'),
            )
            ->first();

        $otherDuration = (int) TimeTrackingEntry::query()
            ->where('timesheet_id', $timesheet->id)
            ->when($existing, fn ($query) => $query->where('id', '!=', $existing->id))
            ->sum('duration');

        if ($otherDuration + (int) $data['duration'] > 10080) {
            throw new RuntimeException('Weekly duration cannot exceed 10080 minutes', 422);
        }

        TimeTrackingEntry::query()->updateOrCreate(
            $match,
            [
                'employee_id' => $timesheet->employee_id,
                'project_id' => $projectId,
                'project_task_id' => $projectTaskId,
                'duration' => $data['duration'],
                'description' => $data['description'] ?? null,
            ],
        );

        if ($timesheet->status === 'rejected') {
            $timesheet->update(['status' => 'open', 'approver_id' => null, 'approved_at' => null]);
        }

        return $timesheet->fresh(['employee', 'approver', 'entries.project', 'entries.projectTask']);
    }

    public function deleteEntry(Timesheet $timesheet, string $entryId): Timesheet
    {
        if (! in_array($timesheet->status, ['open', 'rejected'], true)) {
            throw new RuntimeException('Only open or rejected timesheets can be edited', 409);
        }

        TimeTrackingEntry::query()
            ->where('timesheet_id', $timesheet->id)
            ->where('id', $entryId)
            ->firstOrFail()
            ->delete();

        return $timesheet->fresh(['employee', 'approver', 'entries']);
    }

    public function submit(Timesheet $timesheet): Timesheet
    {
        if (! in_array($timesheet->status, ['open', 'rejected'], true)) {
            throw new RuntimeException('Timesheet cannot be submitted from its current status', 409);
        }

        $timesheet->update(['status' => 'ready_to_submit', 'approver_id' => null, 'approved_at' => null]);

        return $timesheet->fresh(['employee', 'approver', 'entries']);
    }

    public function decide(Timesheet $timesheet, Employee $actor, bool $approve): Timesheet
    {
        if ($timesheet->status !== 'ready_to_submit') {
            throw new RuntimeException('Only submitted timesheets can be reviewed', 409);
        }

        $timesheet->update([
            'status' => $approve ? 'approved' : 'rejected',
            'approver_id' => $actor->id,
            'approved_at' => $approve ? now() : null,
        ]);

        return $timesheet->fresh(['employee', 'approver', 'entries']);
    }

    public function canApprove(Employee $actor, Timesheet $timesheet): bool
    {
        if ($actor->hasAnyRole(['administrator', 'hr'])) {
            return true;
        }

        return DirectReport::query()
            ->where('company_id', $actor->company_id)
            ->where('manager_id', $actor->id)
            ->where('employee_id', $timesheet->employee_id)
            ->exists();
    }

    public function pending(Company $company, Employee $actor): Collection
    {
        $query = Timesheet::query()
            ->with(['employee', 'approver', 'entries'])
            ->where('company_id', $company->id)
            ->where('status', 'ready_to_submit');

        if ($actor->hasAnyRole(['administrator', 'hr'])) {
            $query->whereDate('started_at', '<', now()->startOfWeek()->toDateString())
                ->whereDoesntHave('employee.managerLinks');
        } else {
            $employeeIds = DirectReport::query()
                ->where('company_id', $company->id)
                ->where('manager_id', $actor->id)
                ->pluck('employee_id');
            $query->whereIn('employee_id', $employeeIds);
        }

        return $query->orderBy('started_at')->get();
    }

    public function setWorkFromHome(Company $company, Employee $employee, string $date, bool $enabled): bool
    {
        if (! $company->work_from_home_enabled) {
            throw new RuntimeException('Work from home is disabled for this company', 409);
        }

        if ($enabled) {
            EmployeeWorkFromHome::query()->firstOrCreate([
                'company_id' => $company->id,
                'employee_id' => $employee->id,
                'date' => CarbonImmutable::parse($date)->toDateString(),
            ]);
        } else {
            EmployeeWorkFromHome::query()
                ->where('employee_id', $employee->id)
                ->whereDate('date', $date)
                ->delete();
        }

        return $enabled;
    }

    public function listProjectsForTimesheet(Company $company, Employee $employee): Collection
    {
        return $this->projects->listProjectsForTimesheet($company, $employee);
    }

    public function listTasksForTimesheet(Company $company, Employee $employee, string $projectId): Collection
    {
        return $this->projects->listTasksForTimesheet($company, $employee, $projectId);
    }

    public function currentWeekPayload(Company $company, Employee $employee): array
    {
        return $this->payload($this->createOrGet($company, $employee));
    }

    public function isWorkFromHomeToday(Employee $employee): bool
    {
        return EmployeeWorkFromHome::query()
            ->where('employee_id', $employee->id)
            ->whereDate('date', now()->toDateString())
            ->exists();
    }

    public function payload(Timesheet $timesheet): array
    {
        $timesheet->loadMissing(['employee', 'approver', 'entries.project', 'entries.projectTask']);

        return [
            'id' => (string) $timesheet->id,
            'company_id' => (string) $timesheet->company_id,
            'employee_id' => (string) $timesheet->employee_id,
            'employee' => $timesheet->employee ? $this->employeeSummary($timesheet->employee) : null,
            'started_at' => $timesheet->started_at->toDateString(),
            'ended_at' => $timesheet->ended_at->toDateString(),
            'status' => $timesheet->status,
            'approved_at' => $timesheet->approved_at?->toIso8601String(),
            'approver_id' => $timesheet->approver_id ? (string) $timesheet->approver_id : null,
            'approver_name' => $timesheet->approver?->fullName(),
            'entries' => $timesheet->entries->map(fn (TimeTrackingEntry $entry) => $this->entryPayload($entry))->values()->all(),
            'total_duration' => (int) $timesheet->entries->sum('duration'),
        ];
    }

    private function entryPayload(TimeTrackingEntry $entry): array
    {
        $entry->loadMissing(['project', 'projectTask']);

        return [
            'id' => (string) $entry->id,
            'timesheet_id' => (string) $entry->timesheet_id,
            'employee_id' => (string) $entry->employee_id,
            'project_id' => $entry->project_id ? (string) $entry->project_id : null,
            'project_task_id' => $entry->project_task_id ? (string) $entry->project_task_id : null,
            'project_name' => $entry->project?->name,
            'project_task_title' => $entry->projectTask?->title,
            'duration' => (int) $entry->duration,
            'happened_at' => $entry->happened_at->toDateString(),
            'description' => $entry->description,
        ];
    }

    private function employeeSummary(Employee $employee): array
    {
        return [
            'id' => (string) $employee->id,
            'first_name' => $employee->first_name,
            'last_name' => $employee->last_name,
            'email' => $employee->email,
        ];
    }
}
