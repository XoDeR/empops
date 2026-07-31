<?php

namespace Modules\Recruit\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Employee\Models\Position;
use Modules\Team\Models\Team;

class JobOpening extends Model
{
    use HasUuids;

    protected $fillable = [
        'company_id',
        'position_id',
        'recruiting_stage_template_id',
        'team_id',
        'fulfilled_by_candidate_id',
        'title',
        'description',
        'slug',
        'reference_number',
        'active',
        'fulfilled',
        'page_views',
        'activated_at',
        'fulfilled_at',
    ];

    protected function casts(): array
    {
        return [
            'active' => 'boolean',
            'fulfilled' => 'boolean',
            'page_views' => 'integer',
            'activated_at' => 'datetime',
            'fulfilled_at' => 'datetime',
        ];
    }

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function position(): BelongsTo
    {
        return $this->belongsTo(Position::class);
    }

    public function template(): BelongsTo
    {
        return $this->belongsTo(RecruitingStageTemplate::class, 'recruiting_stage_template_id');
    }

    public function team(): BelongsTo
    {
        return $this->belongsTo(Team::class);
    }

    public function sponsors(): BelongsToMany
    {
        return $this->belongsToMany(Employee::class, 'job_opening_sponsor')
            ->withTimestamps();
    }

    public function candidates(): HasMany
    {
        return $this->hasMany(Candidate::class);
    }

    public function fulfilledBy(): BelongsTo
    {
        return $this->belongsTo(Candidate::class, 'fulfilled_by_candidate_id');
    }
}
