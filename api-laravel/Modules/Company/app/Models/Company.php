<?php

namespace Modules\Company\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Employee\Models\Employee;
use Modules\Employee\Models\EmployeeStatus;
use Modules\Employee\Models\Position;

class Company extends Model
{
    use HasUuids;

    protected $fillable = [
        'name',
        'slug',
        'currency',
        'code_to_join_company',
    ];

    public function employees(): HasMany
    {
        return $this->hasMany(Employee::class);
    }

    public function positions(): HasMany
    {
        return $this->hasMany(Position::class);
    }

    public function employeeStatuses(): HasMany
    {
        return $this->hasMany(EmployeeStatus::class);
    }
}
