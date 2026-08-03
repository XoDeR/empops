<?php

namespace Modules\Company\Services;

use Carbon\Carbon;
use Illuminate\Support\Collection;
use Modules\Company\Models\Company;
use Modules\Company\Models\Flow;
use Modules\Company\Models\FlowAction;
use Modules\Company\Models\FlowActionRun;
use Modules\Company\Models\FlowStep;
use Modules\Employee\Models\Employee;
use Modules\Notification\Services\NotificationService;

final class FlowService
{
    public function __construct(private readonly NotificationService $notifications) {}

    public function list(Company $company): Collection
    {
        return Flow::query()->with('steps.actions')->where('company_id', $company->id)->orderBy('name')->get();
    }

    public function create(Company $company, array $data): Flow
    {
        return Flow::query()->create(['company_id' => $company->id, ...$data])->load('steps.actions');
    }

    public function update(Flow $flow, array $data): Flow
    {
        $flow->fill($data)->save();

        return $flow->fresh('steps.actions');
    }

    public function delete(Flow $flow): void { $flow->delete(); }

    public function addStep(Flow $flow, array $data): FlowStep
    {
        $number = (int) ($data['number'] ?? 0);
        $unit = $data['unit_of_time'] ?? 'days';
        $modifier = $data['modifier'];
        $days = $unit === 'weeks' ? $number * 7 : ($unit === 'months' ? $number * 30 : $number);
        if ($modifier === 'same_day') {
            $real = 0;
        } elseif ($modifier === 'before') {
            $real = -$days;
        } else {
            $real = $days;
        }

        return FlowStep::query()->create([
            'flow_id' => $flow->id,
            'number' => $number,
            'unit_of_time' => $unit,
            'modifier' => $modifier,
            'real_number_of_days' => $real,
        ]);
    }

    public function removeStep(FlowStep $step): void { $step->delete(); }

    public function addAction(FlowStep $step, array $data): FlowAction
    {
        return FlowAction::query()->create(['step_id' => $step->id, ...$data]);
    }

    public function removeAction(FlowAction $action): void { $action->delete(); }

    public function scheduleForEmployee(Company $company, Employee $employee, string $flowType, Carbon $anchorDate): int
    {
        $count = 0;
        $flows = Flow::query()->with('steps.actions')->where('company_id', $company->id)->where('type', $flowType)->get();
        foreach ($flows as $flow) {
            foreach ($flow->steps as $step) {
                $offset = (int) $step->real_number_of_days;
                foreach ($step->actions as $action) {
                    FlowActionRun::query()->firstOrCreate([
                        'company_id' => $company->id,
                        'flow_action_id' => $action->id,
                        'employee_id' => $employee->id,
                        'due_on' => $anchorDate->copy()->addDays($offset)->toDateString(),
                    ]);
                    $count++;
                }
            }
        }

        return $count;
    }

    public function processDueRuns(string|Carbon|null $date = null): int
    {
        $date = $date instanceof Carbon ? $date : Carbon::parse($date ?: now());
        $count = 0;
        FlowActionRun::query()->with(['company', 'employee.managerLinks.manager', 'action'])
            ->whereNull('executed_at')->whereDate('due_on', '<=', $date)->chunkById(100, function ($runs) use (&$count) {
                foreach ($runs as $run) {
                    if ($run->action->type === 'notification') {
                        foreach ($this->recipients($run) as $recipient) {
                            $this->notifications->create($run->company, $recipient, 'flow_action', [
                                'flow_action_run_id' => $run->id,
                                'employee_id' => $run->employee_id,
                                'message' => $run->action->specific_recipient_information,
                            ]);
                        }
                    }
                    $run->update(['executed_at' => now()]);
                    $count++;
                }
            });

        return $count;
    }

    private function recipients(FlowActionRun $run): Collection
    {
        return match ($run->action->recipient) {
            'employee' => collect([$run->employee]),
            'manager' => $run->employee->managerLinks->pluck('manager')->filter()->unique('id')->values(),
            'hr' => Employee::query()->where('company_id', $run->company_id)->where('locked', false)->role('hr')->get(),
            default => collect(),
        };
    }
}
