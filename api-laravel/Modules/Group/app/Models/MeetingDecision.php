<?php

namespace Modules\Group\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class MeetingDecision extends Model
{
    use HasUuids;
    protected $fillable = ['agenda_item_id', 'description'];
    public function agendaItem(): BelongsTo { return $this->belongsTo(AgendaItem::class); }
}
