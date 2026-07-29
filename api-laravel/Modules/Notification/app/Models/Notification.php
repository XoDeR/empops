<?php

namespace Modules\Notification\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;

class Notification extends Model
{
    use HasUuids;

    protected $fillable = [
        'company_id',
        'employee_id',
        'action',
        'objects',
        'read_at',
    ];

    protected function casts(): array
    {
        return [
            'objects' => 'array',
            'read_at' => 'datetime',
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
