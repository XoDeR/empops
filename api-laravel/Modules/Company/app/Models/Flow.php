<?php

namespace Modules\Company\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Flow extends Model
{
    use HasUuids;
    protected $fillable = ['company_id', 'name', 'type'];
    public function company(): BelongsTo { return $this->belongsTo(Company::class); }
    public function steps(): HasMany { return $this->hasMany(FlowStep::class)->orderBy('real_number_of_days'); }
}
