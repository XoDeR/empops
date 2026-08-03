<?php

namespace Modules\Company\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Employee\Models\Employee;

class AskMeAnythingQuestion extends Model
{
    use HasUuids;
    protected $fillable = ['ask_me_anything_session_id', 'employee_id', 'question', 'answered', 'anonymous'];
    protected function casts(): array { return ['answered' => 'boolean', 'anonymous' => 'boolean']; }
    public function session(): BelongsTo { return $this->belongsTo(AskMeAnythingSession::class, 'ask_me_anything_session_id'); }
    public function employee(): BelongsTo { return $this->belongsTo(Employee::class); }
}
