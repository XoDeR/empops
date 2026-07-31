<?php

namespace Modules\Project\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class ProjectBoard extends Model
{
    use HasUuids;

    protected $fillable = [
        'project_id',
        'name',
    ];

    public function project(): BelongsTo
    {
        return $this->belongsTo(Project::class);
    }

    public function sprints(): HasMany
    {
        return $this->hasMany(ProjectSprint::class, 'project_board_id');
    }

    public function issues(): HasMany
    {
        return $this->hasMany(ProjectIssue::class, 'project_board_id');
    }
}
