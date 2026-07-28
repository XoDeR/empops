<?php

namespace Modules\Employee\Models;

use App\Models\User;
use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Company\Models\Company;
use Modules\Team\Models\Team;
use Spatie\Permission\Traits\HasRoles;

class Employee extends Model
{
    use HasRoles;
    use HasUuids;

    protected $guard_name = 'web';

    protected $fillable = [
        'company_id',
        'user_id',
        'email',
        'first_name',
        'last_name',
        'hired_at',
        'position_id',
        'employee_status_id',
        'invitation_link',
        'invitation_used_at',
        'locked',
    ];

    protected function casts(): array
    {
        return [
            'hired_at' => 'date',
            'invitation_used_at' => 'datetime',
            'locked' => 'boolean',
        ];
    }

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }

    public function position(): BelongsTo
    {
        return $this->belongsTo(Position::class);
    }

    public function status(): BelongsTo
    {
        return $this->belongsTo(EmployeeStatus::class, 'employee_status_id');
    }

    public function teams(): BelongsToMany
    {
        return $this->belongsToMany(Team::class, 'employee_team')->withTimestamps();
    }

    public function managedReports(): HasMany
    {
        return $this->hasMany(DirectReport::class, 'manager_id');
    }

    public function managerLinks(): HasMany
    {
        return $this->hasMany(DirectReport::class, 'employee_id');
    }

    public function fullName(): string
    {
        return trim($this->first_name.' '.$this->last_name);
    }
}
