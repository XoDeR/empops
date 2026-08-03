<?php

namespace Modules\Time\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Employee\Models\Employee;

class EmployeeDailyCalendarEntry extends Model
{
    use HasUuids;

    protected $fillable = ['employee_id', 'log_date', 'new_balance', 'daily_accrued_amount', 'current_holidays_per_year', 'default_amount_of_allowed_holidays_in_company'];

    protected function casts(): array
    {
        return [
            'log_date' => 'date',
            'new_balance' => 'decimal:4',
            'daily_accrued_amount' => 'decimal:6',
            'current_holidays_per_year' => 'decimal:2',
            'default_amount_of_allowed_holidays_in_company' => 'decimal:2',
        ];
    }

    public function employee(): BelongsTo { return $this->belongsTo(Employee::class); }
}
