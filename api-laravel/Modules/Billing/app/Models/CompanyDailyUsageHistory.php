<?php

namespace Modules\Billing\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Company\Models\Company;

class CompanyDailyUsageHistory extends Model
{
    use HasUuids;
    protected $table = 'company_daily_usage_history';
    protected $fillable = ['company_id', 'number_of_active_employees', 'logged_on'];
    protected function casts(): array { return ['number_of_active_employees' => 'integer', 'logged_on' => 'date']; }
    public function company(): BelongsTo { return $this->belongsTo(Company::class); }
    public function details(): HasMany { return $this->hasMany(CompanyUsageHistoryDetail::class, 'usage_history_id'); }
}
