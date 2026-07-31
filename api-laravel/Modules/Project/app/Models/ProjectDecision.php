<?php

namespace Modules\Project\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Modules\Employee\Models\Employee;

class ProjectDecision extends Model
{
    use HasUuids;

    protected $fillable = [
        'project_id',
        'author_id',
        'title',
        'decided_at',
    ];

    protected function casts(): array
    {
        return [
            'decided_at' => 'date',
        ];
    }

    public function project(): BelongsTo
    {
        return $this->belongsTo(Project::class);
    }

    public function author(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'author_id');
    }

    public function deciders(): BelongsToMany
    {
        return $this->belongsToMany(Employee::class, 'project_decision_deciders', 'project_decision_id', 'employee_id')
            ->withTimestamps();
    }
}
