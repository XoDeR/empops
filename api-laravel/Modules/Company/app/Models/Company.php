<?php

namespace Modules\Company\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Employee\Models\Employee;
use Modules\Employee\Models\EmployeeStatus;
use Modules\Employee\Models\Position;
use Modules\Recruit\Models\JobOpening;
use Spatie\MediaLibrary\HasMedia;
use Spatie\MediaLibrary\InteractsWithMedia;

class Company extends Model implements HasMedia
{
    use HasUuids;
    use InteractsWithMedia;

    protected $fillable = [
        'name',
        'slug',
        'currency',
        'code_to_join_company',
        'work_from_home_enabled',
        'e_coffee_enabled',
    ];

    protected function casts(): array
    {
        return [
            'work_from_home_enabled' => 'boolean',
            'e_coffee_enabled' => 'boolean',
        ];
    }

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

    public function jobOpenings(): HasMany
    {
        return $this->hasMany(JobOpening::class);
    }

    public function jobOpeningsPublic(): HasMany
    {
        return $this->hasMany(JobOpening::class)
            ->where('active', true)
            ->where('fulfilled', false);
    }

    public function registerMediaCollections(): void
    {
        $this->addMediaCollection('logo')
            ->singleFile()
            ->acceptsMimeTypes(['image/jpeg', 'image/png', 'image/gif', 'image/webp']);
    }
}
