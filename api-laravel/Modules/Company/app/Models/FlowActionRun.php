<?php

namespace Modules\Company\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Employee\Models\Employee;

class FlowActionRun extends Model
{
    use HasUuids;
    protected $fillable = ['company_id', 'flow_action_id', 'employee_id', 'due_on', 'executed_at'];
    protected function casts(): array { return ['due_on' => 'date', 'executed_at' => 'datetime']; }
    public function company(): BelongsTo { return $this->belongsTo(Company::class); }
    public function action(): BelongsTo { return $this->belongsTo(FlowAction::class, 'flow_action_id'); }
    public function employee(): BelongsTo { return $this->belongsTo(Employee::class); }
}
