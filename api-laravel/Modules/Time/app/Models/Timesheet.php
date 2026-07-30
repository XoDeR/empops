<?php

namespace Modules\Time\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;

class Timesheet extends Model
{
    use HasUuids;

    protected $fillable = [
        'company_id',
        'employee_id',
        'started_at',
        'ended_at',
        'status',
        'approver_id',
        'approved_at',
    ];

    protected function casts(): array
    {
        return [
            'started_at' => 'date',
            'ended_at' => 'date',
            'approved_at' => 'datetime',
        ];
    }

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function employee(): BelongsTo
    {
        return $this->belongsTo(Employee::class);
    }

    public function approver(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'approver_id');
    }

    public function entries(): HasMany
    {
        return $this->hasMany(TimeTrackingEntry::class)->orderBy('happened_at');
    }
}
