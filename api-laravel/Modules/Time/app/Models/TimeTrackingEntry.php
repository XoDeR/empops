<?php

namespace Modules\Time\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Employee\Models\Employee;

class TimeTrackingEntry extends Model
{
    use HasUuids;

    protected $fillable = [
        'timesheet_id',
        'employee_id',
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
}
