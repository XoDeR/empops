<?php

namespace Modules\Project\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Team\Models\Team;
use Spatie\MediaLibrary\HasMedia;
use Spatie\MediaLibrary\InteractsWithMedia;

class Project extends Model implements HasMedia
{
    use HasUuids;
    use InteractsWithMedia;

    public const STATUS_CREATED = 'created';

    public const STATUS_STARTED = 'started';

    public const STATUS_PAUSED = 'paused';

    public const STATUS_CANCELLED = 'cancelled';

    public const STATUS_CLOSED = 'closed';

    protected $fillable = [
        'company_id',
        'project_lead_id',
        'status',
        'completed',
        'name',
        'code',
        'short_code',
        'emoji',
        'summary',
        'description',
        'started_at',
        'planned_finished_at',
        'actually_finished_at',
    ];

    protected function casts(): array
    {
        return [
            'completed' => 'boolean',
            'started_at' => 'datetime',
            'planned_finished_at' => 'datetime',
            'actually_finished_at' => 'datetime',
        ];
    }

    public function registerMediaCollections(): void
    {
        $this->addMediaCollection('files');
    }

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function lead(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'project_lead_id');
    }

    public function employees(): BelongsToMany
    {
        return $this->belongsToMany(Employee::class, 'employee_project')
            ->withPivot('role')
            ->withTimestamps();
    }

    public function teams(): BelongsToMany
    {
        return $this->belongsToMany(Team::class, 'project_team')
            ->withTimestamps();
    }

    public function links(): HasMany
    {
        return $this->hasMany(ProjectLink::class);
    }

    public function statuses(): HasMany
    {
        return $this->hasMany(ProjectStatus::class);
    }

    public function messages(): HasMany
    {
        return $this->hasMany(ProjectMessage::class);
    }

    public function decisions(): HasMany
    {
        return $this->hasMany(ProjectDecision::class);
    }

    public function tasks(): HasMany
    {
        return $this->hasMany(ProjectTask::class);
    }

    public function lists(): HasMany
    {
        return $this->hasMany(ProjectTaskList::class, 'project_id');
    }

    public function boards(): HasMany
    {
        return $this->hasMany(ProjectBoard::class);
    }

    public function sprints(): HasMany
    {
        return $this->hasMany(ProjectSprint::class);
    }

    public function issues(): HasMany
    {
        return $this->hasMany(ProjectIssue::class);
    }

    public function labels(): HasMany
    {
        return $this->hasMany(ProjectLabel::class);
    }
}
