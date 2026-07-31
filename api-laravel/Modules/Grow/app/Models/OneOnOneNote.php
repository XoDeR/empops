<?php

namespace Modules\Grow\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class OneOnOneNote extends Model
{
    use HasUuids;

    protected $fillable = [
        'one_on_one_entry_id',
        'note',
    ];

    public function entry(): BelongsTo
    {
        return $this->belongsTo(OneOnOneEntry::class, 'one_on_one_entry_id');
    }
}
