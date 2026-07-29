<?php

namespace Modules\Employee\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Company\Models\Company;

class Worklog extends Model
{
    use HasUuids;

    protected $fillable = [
        'company_id',
        'employee_id',
        'content',
        'logged_on',
    ];

    protected function casts(): array
    {
        return [
            'logged_on' => 'date',
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
