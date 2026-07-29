<?php

namespace Modules\Place\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Country extends Model
{
    use HasUuids;

    protected $fillable = [
        'name',
        'code',
    ];

    public function places(): HasMany
    {
        return $this->hasMany(Place::class);
    }
}
