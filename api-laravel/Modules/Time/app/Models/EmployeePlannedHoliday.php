<?php

namespace Modules\Time\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Employee\Models\Employee;

class EmployeePlannedHoliday extends Model
{
    use HasUuids;

    protected $fillable = ['employee_id', 'planned_date', 'type', 'full', 'actually_taken'];

    protected function casts(): array
    {
        return ['planned_date' => 'date', 'full' => 'boolean', 'actually_taken' => 'boolean'];
    }

    public function employee(): BelongsTo { return $this->belongsTo(Employee::class); }
}
