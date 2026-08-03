<?php

namespace Modules\Group\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Modules\Employee\Models\Employee;

class Meeting extends Model
{
    use HasUuids;
    protected $fillable = ['group_id', 'happened', 'happened_at'];
    protected function casts(): array { return ['happened' => 'boolean', 'happened_at' => 'date']; }
    public function group(): BelongsTo { return $this->belongsTo(Group::class); }
    public function attendees(): BelongsToMany { return $this->belongsToMany(Employee::class, 'employee_meeting')->withPivot(['was_a_guest', 'attended'])->withTimestamps(); }
    public function agendaItems(): HasMany { return $this->hasMany(AgendaItem::class)->orderBy('position'); }
}
