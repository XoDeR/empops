<?php

namespace Modules\Recruit\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Spatie\MediaLibrary\HasMedia;
use Spatie\MediaLibrary\InteractsWithMedia;

class Candidate extends Model implements HasMedia
{
    use HasUuids;
    use InteractsWithMedia;

    protected $fillable = [
        'company_id',
        'job_opening_id',
        'employee_id',
        'name',
        'email',
        'uuid',
        'url',
        'desired_salary',
        'notes',
        'application_completed',
        'rejected',
        'employee_name',
    ];

    protected function casts(): array
    {
        return [
            'application_completed' => 'boolean',
            'rejected' => 'boolean',
        ];
    }

    public function registerMediaCollections(): void
    {
        $this->addMediaCollection('cv');
    }

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function jobOpening(): BelongsTo
    {
        return $this->belongsTo(JobOpening::class);
    }

    public function employee(): BelongsTo
    {
        return $this->belongsTo(Employee::class);
    }

    public function stages(): HasMany
    {
        return $this->hasMany(CandidateStage::class)->orderBy('stage_position');
    }
}
