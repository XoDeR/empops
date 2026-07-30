<?php

namespace Modules\Finance\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;

class Expense extends Model
{
    use HasUuids;

    protected $fillable = [
        'company_id',
        'employee_id',
        'expense_category_id',
        'status',
        'title',
        'amount',
        'currency',
        'converted_amount',
        'converted_to_currency',
        'converted_at',
        'exchange_rate',
        'description',
        'expensed_at',
        'manager_approver_id',
        'manager_approver_approved_at',
        'manager_rejection_explanation',
        'accounting_approver_id',
        'accounting_approver_approved_at',
        'accounting_rejection_explanation',
    ];

    protected function casts(): array
    {
        return [
            'amount' => 'integer',
            'converted_amount' => 'integer',
            'exchange_rate' => 'decimal:8',
            'converted_at' => 'datetime',
            'expensed_at' => 'date',
            'manager_approver_approved_at' => 'datetime',
            'accounting_approver_approved_at' => 'datetime',
        ];
    }

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function employee(): BelongsTo
    {
        return $this->belongsTo(Employee::class);
    }

    public function category(): BelongsTo
    {
        return $this->belongsTo(ExpenseCategory::class, 'expense_category_id');
    }

    public function managerApprover(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'manager_approver_id');
    }

    public function accountingApprover(): BelongsTo
    {
        return $this->belongsTo(Employee::class, 'accounting_approver_id');
    }
}
