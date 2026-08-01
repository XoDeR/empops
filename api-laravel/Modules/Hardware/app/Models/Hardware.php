<?php

namespace Modules\Hardware\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;

class Hardware extends Model
{
    use HasUuids;

    protected $table = 'hardware';

    protected $fillable = [
        'company_id',
        'employee_id',
        'name',
        'serial_number',
    ];

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function employee(): BelongsTo
    {
        return $this->belongsTo(Employee::class);
    }
}
