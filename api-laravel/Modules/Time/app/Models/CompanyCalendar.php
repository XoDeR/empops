<?php

namespace Modules\Time\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class CompanyCalendar extends Model
{
    use HasUuids;

    protected $fillable = ['company_pto_policy_id', 'day', 'day_of_week', 'day_of_year', 'is_worked'];

    protected function casts(): array
    {
        return ['day' => 'date', 'day_of_week' => 'integer', 'day_of_year' => 'integer', 'is_worked' => 'boolean'];
    }

    public function policy(): BelongsTo { return $this->belongsTo(CompanyPtoPolicy::class, 'company_pto_policy_id'); }
}
