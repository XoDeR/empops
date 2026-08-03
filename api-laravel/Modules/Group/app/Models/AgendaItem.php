<?php

namespace Modules\Group\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Employee\Models\Employee;

class AgendaItem extends Model
{
    use HasUuids;
    protected $fillable = ['meeting_id', 'position', 'checked', 'summary', 'description', 'presented_by_id'];
    protected function casts(): array { return ['position' => 'integer', 'checked' => 'boolean']; }
    public function meeting(): BelongsTo { return $this->belongsTo(Meeting::class); }
    public function presenter(): BelongsTo { return $this->belongsTo(Employee::class, 'presented_by_id'); }
    public function decisions(): HasMany { return $this->hasMany(MeetingDecision::class); }
}
