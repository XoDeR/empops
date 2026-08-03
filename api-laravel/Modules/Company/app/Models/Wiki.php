<?php

namespace Modules\Company\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Wiki extends Model
{
    use HasUuids;
    protected $fillable = ['company_id', 'title'];
    public function company(): BelongsTo { return $this->belongsTo(Company::class); }
    public function pages(): HasMany { return $this->hasMany(WikiPage::class); }
}
