<?php

namespace Modules\Company\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class FlowStep extends Model
{
    use HasUuids;
    protected $fillable = ['flow_id', 'number', 'unit_of_time', 'modifier', 'real_number_of_days'];
    protected function casts(): array { return ['number' => 'integer', 'real_number_of_days' => 'integer']; }
    public function flow(): BelongsTo { return $this->belongsTo(Flow::class); }
    public function actions(): HasMany { return $this->hasMany(FlowAction::class); }
}
