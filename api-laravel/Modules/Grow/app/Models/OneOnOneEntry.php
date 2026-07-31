<?php

namespace Modules\Grow\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;

class OneOnOneEntry extends Model
{
    use HasUuids;

    protected $fillable = [
        'company_id',
        'manager_id',
        'employee_id',
        'happened',
        'happened_at',
    ];

    protected function casts(): array
    {
        return [
            'happened' => 'boolean',
            'happened_at' => 'datetime',
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

    public function employee(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'employee_id');
    }

    public function talkingPoints(): HasMany
    {
        return $this->hasMany(OneOnOneTalkingPoint::class);
    }

    public function actionItems(): HasMany
    {
        return $this->hasMany(OneOnOneActionItem::class);
    }

    public function notes(): HasMany
    {
        return $this->hasMany(OneOnOneNote::class);
    }
}
