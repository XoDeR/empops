<?php

namespace Modules\Recruit\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Employee\Models\Employee;

class CandidateStage extends Model
{
    use HasUuids;

    protected $fillable = [
        'candidate_id',
        'decider_id',
        'stage_name',
        'stage_position',
        'status',
        'decider_name',
        'decided_at',
    ];

    protected function casts(): array
    {
        return [
            'stage_position' => 'integer',
            'decided_at' => 'datetime',
        ];
    }

    public function candidate(): BelongsTo
    {
        return $this->belongsTo(Candidate::class);
    }

    public function decider(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'decider_id');
    }

    public function notes(): HasMany
    {
        return $this->hasMany(CandidateStageNote::class);
    }

    public function participants(): HasMany
    {
        return $this->hasMany(CandidateStageParticipant::class);
    }
}
