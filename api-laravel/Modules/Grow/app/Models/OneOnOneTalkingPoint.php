<?php

namespace Modules\Grow\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class OneOnOneTalkingPoint extends Model
{
    use HasUuids;

    protected $fillable = [
        'one_on_one_entry_id',
        'description',
        'checked',
    ];

    protected function casts(): array
    {
        return [
            'checked' => 'boolean',
        ];
    }

    public function entry(): BelongsTo
    {
        return $this->belongsTo(OneOnOneEntry::class, 'one_on_one_entry_id');
    }
}
