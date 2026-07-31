<?php

namespace Modules\Time\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Employee\Models\Employee;
use Modules\Project\Models\Project;
use Modules\Project\Models\ProjectTask;

class TimeTrackingEntry extends Model
{
    use HasUuids;

    protected $fillable = [
        'timesheet_id',
        'employee_id',
        'project_id',
        'project_task_id',
        'duration',
        'happened_at',
        'description',
    ];

    protected function casts(): array
    {
        return [
            'duration' => 'integer',
            'happened_at' => 'date',
        ];
    }

    public function timesheet(): BelongsTo
    {
        return $this->belongsTo(Timesheet::class);
    }

    public function employee(): BelongsTo
    {
        return $this->belongsTo(Employee::class);
    }

    public function project(): BelongsTo
    {
        return $this->belongsTo(Project::class);
    }

    public function projectTask(): BelongsTo
    {
        return $this->belongsTo(ProjectTask::class, 'project_task_id');
    }
}
