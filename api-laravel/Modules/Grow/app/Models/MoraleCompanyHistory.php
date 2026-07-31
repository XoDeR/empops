<?php

namespace Modules\Grow\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Company\Models\Company;

class MoraleCompanyHistory extends Model
{
    use HasUuids;

    protected $fillable = [
        'company_id',
        'average',
        'number_of_employees',
    ];

    protected function casts(): array
    {
        return [
            'average' => 'float',
            'number_of_employees' => 'integer',
        ];
    }

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }
}
