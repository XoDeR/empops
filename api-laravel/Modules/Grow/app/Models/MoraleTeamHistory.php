<?php

namespace Modules\Grow\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Team\Models\Team;

class MoraleTeamHistory extends Model
{
    use HasUuids;

    protected $fillable = [
        'team_id',
        'average',
        'number_of_team_members',
    ];

    protected function casts(): array
    {
        return [
            'average' => 'float',
            'number_of_team_members' => 'integer',
        ];
    }

    public function team(): BelongsTo
    {
        return $this->belongsTo(Team::class);
    }
}
