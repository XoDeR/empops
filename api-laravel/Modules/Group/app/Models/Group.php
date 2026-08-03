<?php

namespace Modules\Group\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;

class Group extends Model
{
    use HasUuids;
    protected $fillable = ['company_id', 'name', 'mission'];
    public function company(): BelongsTo { return $this->belongsTo(Company::class); }
    public function members(): BelongsToMany { return $this->belongsToMany(Employee::class, 'employee_group')->withTimestamps(); }
    public function meetings(): HasMany { return $this->hasMany(Meeting::class); }
}
