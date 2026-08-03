<?php

namespace Modules\Billing\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Employee\Models\Employee;

class CompanyUsageHistoryDetail extends Model
{
    use HasUuids;
    protected $fillable = ['usage_history_id', 'employee_id', 'employee_name', 'employee_email'];
    public function usage(): BelongsTo { return $this->belongsTo(CompanyDailyUsageHistory::class, 'usage_history_id'); }
    public function employee(): BelongsTo { return $this->belongsTo(Employee::class); }
}
