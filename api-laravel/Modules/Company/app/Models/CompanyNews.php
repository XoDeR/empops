<?php

namespace Modules\Company\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Employee\Models\Employee;

class CompanyNews extends Model
{
    use HasUuids;

    protected $table = 'company_news';

    protected $fillable = [
        'company_id',
        'author_id',
        'author_name',
        'title',
        'content',
    ];

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function author(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'author_id');
    }
}
