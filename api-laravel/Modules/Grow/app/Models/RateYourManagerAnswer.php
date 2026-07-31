<?php

namespace Modules\Grow\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Employee\Models\Employee;

class RateYourManagerAnswer extends Model
{
    use HasUuids;

    protected $fillable = [
        'rate_your_manager_survey_id',
        'employee_id',
        'active',
        'rating',
        'comment',
        'reveal_identity_to_manager',
    ];

    protected function casts(): array
    {
        return [
            'active' => 'boolean',
            'reveal_identity_to_manager' => 'boolean',
        ];
    }

    public function survey(): BelongsTo
    {
        return $this->belongsTo(RateYourManagerSurvey::class, 'rate_your_manager_survey_id');
    }

    public function employee(): BelongsTo
    {
        return $this->belongsTo(Employee::class);
    }
}
