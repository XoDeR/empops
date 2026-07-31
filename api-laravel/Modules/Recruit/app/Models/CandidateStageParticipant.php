<?php

namespace Modules\Recruit\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Employee\Models\Employee;

class CandidateStageParticipant extends Model
{
    use HasUuids;

    protected $fillable = [
        'candidate_stage_id',
        'participant_id',
        'participant_name',
        'participated',
        'participated_at',
    ];

    protected function casts(): array
    {
        return [
            'participated' => 'boolean',
            'participated_at' => 'datetime',
        ];
    }

    public function stage(): BelongsTo
    {
        return $this->belongsTo(CandidateStage::class, 'candidate_stage_id');
    }

    public function participant(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'participant_id');
    }
}
