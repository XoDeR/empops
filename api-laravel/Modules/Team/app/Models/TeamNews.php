<?php

namespace Modules\Team\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;

class TeamNews extends Model
{
    use HasUuids;

    protected $table = 'team_news';

    protected $fillable = [
        'company_id',
        'team_id',
        'author_id',
        'author_name',
        'title',
        'content',
    ];

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function team(): BelongsTo
    {
        return $this->belongsTo(Team::class);
    }

    public function author(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'author_id');
    }
}
