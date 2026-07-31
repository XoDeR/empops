<?php

namespace Modules\Recruit\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Company\Models\Company;

class RecruitingStageTemplate extends Model
{
    use HasUuids;

    protected $fillable = [
        'company_id',
        'name',
    ];

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function stages(): HasMany
    {
        return $this->hasMany(RecruitingStage::class)->orderBy('position');
    }

    public function jobOpenings(): HasMany
    {
        return $this->hasMany(JobOpening::class, 'recruiting_stage_template_id');
    }
}