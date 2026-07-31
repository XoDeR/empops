<?php

namespace Modules\Recruit\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class RecruitingStage extends Model
{
    use HasUuids;

    protected $fillable = [
        'recruiting_stage_template_id',
        'name',
        'position',
    ];

    protected function casts(): array
    {
        return [
            'position' => 'integer',
        ];
    }

    public function template(): BelongsTo
    {
        return $this->belongsTo(RecruitingStageTemplate::class, 'recruiting_stage_template_id');
    }
}
