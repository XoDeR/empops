<?php

namespace Modules\Grow\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Employee\Models\Employee;

class ECoffeeMatch extends Model
{
    use HasUuids;

    protected $fillable = [
        'e_coffee_id',
        'employee_id',
        'with_employee_id',
        'happened',
    ];

    protected function casts(): array
    {
        return [
            'happened' => 'boolean',
        ];
    }

    public function eCoffee(): BelongsTo
    {
        return $this->belongsTo(ECoffee::class, 'e_coffee_id');
    }

    public function employee(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'employee_id');
    }

    public function withEmployee(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'with_employee_id');
    }
}
