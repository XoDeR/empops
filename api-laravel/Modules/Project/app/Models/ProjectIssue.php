<?php

namespace Modules\Project\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Modules\Employee\Models\Employee;

class ProjectIssue extends Model
{
    use HasUuids;

    protected $fillable = [
        'project_id',
        'project_board_id',
        'reporter_id',
        'issue_type_id',
        'is_separator',
        'id_in_project',
        'key',
        'slug',
        'title',
        'description',
        'story_points',
    ];

    protected function casts(): array
    {
        return [
            'is_separator' => 'boolean',
            'id_in_project' => 'integer',
            'story_points' => 'integer',
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

    public function reporter(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'reporter_id');
    }

    public function type(): BelongsTo
    {
        return $this->belongsTo(IssueType::class, 'issue_type_id');
    }

    public function sprints(): BelongsToMany
    {
        return $this->belongsToMany(ProjectSprint::class, 'project_issue_project_sprint', 'project_issue_id', 'project_sprint_id')
            ->withPivot('position');
    }

    public function assignees(): BelongsToMany
    {
        return $this->belongsToMany(Employee::class, 'project_issue_assignees', 'project_issue_id', 'employee_id');
    }

    public function labels(): BelongsToMany
    {
        return $this->belongsToMany(ProjectLabel::class, 'project_issue_project_label', 'project_issue_id', 'project_label_id');
    }
}
