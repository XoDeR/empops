<?php

namespace Modules\Company\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Employee\Models\Employee;

class WikiPageRevision extends Model
{
    use HasUuids;
    protected $fillable = ['page_id', 'employee_id', 'employee_name', 'title', 'content'];
    public function page(): BelongsTo { return $this->belongsTo(WikiPage::class, 'page_id'); }
    public function employee(): BelongsTo { return $this->belongsTo(Employee::class); }
}
