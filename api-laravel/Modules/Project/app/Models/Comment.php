<?php

namespace Modules\Project\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\MorphTo;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;

class Comment extends Model
{
    use HasUuids;

    protected $fillable = [
        'company_id',
        'author_id',
        'author_name',
        'content',
        'commentable_id',
        'commentable_type',
    ];

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function author(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'author_id');
    }

    public function commentable(): MorphTo
    {
        return $this->morphTo();
    }
}
