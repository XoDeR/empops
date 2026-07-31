<?php

namespace Modules\Grow\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Employee\Models\Employee;
use Spatie\MediaLibrary\HasMedia;
use Spatie\MediaLibrary\InteractsWithMedia;

class DisciplineEvent extends Model implements HasMedia
{
    use HasUuids;
    use InteractsWithMedia;

    protected $fillable = [
        'discipline_case_id',
        'author_id',
        'author_name',
        'happened_at',
        'description',
    ];

    protected function casts(): array
    {
        return [
            'happened_at' => 'date',
        ];
    }

    public function registerMediaCollections(): void
    {
        $this->addMediaCollection('discipline');
    }

    public function disciplineCase(): BelongsTo
    {
        return $this->belongsTo(DisciplineCase::class);
    }

    public function author(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'author_id');
    }
}
