<?php

namespace Modules\Team\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;

class Team extends Model
{
    use HasUuids;

    protected $fillable = [
        'company_id',
        'name',
        'description',
        'team_leader_id',
    ];

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function leader(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'team_leader_id');
    }

    public function employees(): BelongsToMany
    {
        return $this->belongsToMany(Employee::class, 'employee_team')
            ->withTimestamps();
    }
}
