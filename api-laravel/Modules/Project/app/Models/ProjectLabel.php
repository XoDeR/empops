<?php

namespace Modules\Project\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;

class ProjectLabel extends Model
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

    public function issues(): BelongsToMany
    {
        return $this->belongsToMany(ProjectIssue::class, 'project_issue_project_label', 'project_label_id', 'project_issue_id');
    }
}
