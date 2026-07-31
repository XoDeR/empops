<?php

namespace Modules\Project\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Company\Models\Company;

class IssueType extends Model
{
    use HasUuids;

    protected $fillable = [
        'company_id',
        'name',
        'icon',
    ];

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function issues(): HasMany
    {
        return $this->hasMany(ProjectIssue::class, 'issue_type_id');
    }
}
