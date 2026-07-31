<?php

namespace Modules\Project\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;

class ProjectSprint extends Model
{
    use HasUuids;

    protected $fillable = [
        'project_id',
        'project_board_id',
        'name',
        'active',
        'position',
        'started_at',
        'completed_at',
    ];

    protected function casts(): array
    {
        return [
            'active' => 'boolean',
            'position' => 'integer',
            'started_at' => 'datetime',
            'completed_at' => 'datetime',
        ];
    }

    public function project(): BelongsTo
    {
        return $this->belongsTo(Project::class);
    }

    public function board(): BelongsTo
    {
        return $this->belongsTo(ProjectBoard::class, 'project_board_id');
    }

    public function issues(): BelongsToMany
    {
        return $this->belongsToMany(ProjectIssue::class, 'project_issue_project_sprint', 'project_sprint_id', 'project_issue_id')
            ->withPivot('position')
            ->orderByPivot('position');
    }
}
