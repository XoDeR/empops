<?php

namespace Modules\Grow\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;

class Morale extends Model
{
    use HasUuids;

    protected $fillable = [
        'company_id',
        'employee_id',
        'emotion',
        'comment',
    ];

    protected function casts(): array
    {
        return [
            'emotion' => 'integer',
        ];
    }

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function employee(): BelongsTo
    {
        return $this->belongsTo(Employee::class);
    }
}
