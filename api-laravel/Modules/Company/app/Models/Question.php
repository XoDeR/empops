<?php

namespace Modules\Company\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Question extends Model
{
    use HasUuids;

    protected $fillable = [
        'company_id',
        'title',
        'active',
        'activated_at',
        'deactivated_at',
    ];

    protected function casts(): array
    {
        return [
            'active' => 'boolean',
            'activated_at' => 'datetime',
            'deactivated_at' => 'datetime',
        ];
    }

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function answers(): HasMany
    {
        return $this->hasMany(Answer::class);
    }
}
