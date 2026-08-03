<?php

namespace Modules\Company\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class AskMeAnythingSession extends Model
{
    use HasUuids;
    protected $fillable = ['company_id', 'happened_at', 'active', 'theme'];
    protected function casts(): array { return ['happened_at' => 'date', 'active' => 'boolean']; }
    public function company(): BelongsTo { return $this->belongsTo(Company::class); }
    public function questions(): HasMany { return $this->hasMany(AskMeAnythingQuestion::class); }
}
