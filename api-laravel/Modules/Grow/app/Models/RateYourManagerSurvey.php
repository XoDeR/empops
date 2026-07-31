<?php

namespace Modules\Grow\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;

class RateYourManagerSurvey extends Model
{
    use HasUuids;

    protected $fillable = [
        'company_id',
        'manager_id',
        'active',
        'valid_until_at',
    ];

    protected function casts(): array
    {
        return [
            'active' => 'boolean',
            'valid_until_at' => 'datetime',
        ];
    }

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function manager(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'manager_id');
    }

    public function answers(): HasMany
    {
        return $this->hasMany(RateYourManagerAnswer::class);
    }
}
