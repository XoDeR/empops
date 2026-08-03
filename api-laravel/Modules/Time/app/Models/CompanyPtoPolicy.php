<?php

namespace Modules\Time\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Company\Models\Company;

class CompanyPtoPolicy extends Model
{
    use HasUuids;

    protected $fillable = ['company_id', 'year', 'total_worked_days', 'default_amount_of_allowed_holidays', 'default_amount_of_sick_days', 'default_amount_of_pto_days'];

    protected function casts(): array
    {
        return [
            'year' => 'integer',
            'total_worked_days' => 'integer',
            'default_amount_of_allowed_holidays' => 'decimal:2',
            'default_amount_of_sick_days' => 'decimal:2',
            'default_amount_of_pto_days' => 'decimal:2',
        ];
    }

    public function company(): BelongsTo { return $this->belongsTo(Company::class); }
    public function calendar(): HasMany { return $this->hasMany(CompanyCalendar::class)->orderBy('day'); }
}
