<?php

namespace Modules\Recruit\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Employee\Models\Employee;

class CandidateStageNote extends Model
{
    use HasUuids;

    protected $fillable = [
        'candidate_stage_id',
        'author_id',
        'author_name',
        'note',
    ];

    public function stage(): BelongsTo
    {
        return $this->belongsTo(CandidateStage::class, 'candidate_stage_id');
    }

    public function author(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'author_id');
    }
}
