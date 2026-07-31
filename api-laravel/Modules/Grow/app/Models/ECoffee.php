<?php

namespace Modules\Grow\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Company\Models\Company;

class ECoffee extends Model
{
    use HasUuids;

    protected $table = 'e_coffees';

    protected $fillable = [
        'company_id',
        'batch_number',
        'active',
    ];

    protected function casts(): array
    {
        return [
            'batch_number' => 'integer',
            'active' => 'boolean',
        ];
    }

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function matches(): HasMany
    {
        return $this->hasMany(ECoffeeMatch::class, 'e_coffee_id');
    }
}
