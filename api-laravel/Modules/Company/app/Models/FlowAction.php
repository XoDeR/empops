<?php

namespace Modules\Company\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class FlowAction extends Model
{
    use HasUuids;
    protected $fillable = ['step_id', 'type', 'recipient', 'specific_recipient_information'];
    public function step(): BelongsTo { return $this->belongsTo(FlowStep::class); }
    public function runs(): HasMany { return $this->hasMany(FlowActionRun::class); }
}
