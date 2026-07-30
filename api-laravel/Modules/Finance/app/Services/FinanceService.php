<?php

namespace Modules\Finance\Services;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\DB;
use Modules\Company\Models\Company;
use Modules\Employee\Models\DirectReport;
use Modules\Employee\Models\Employee;
use Modules\Finance\Models\Expense;
use Modules\Finance\Models\ExpenseCategory;
use Modules\Notification\Services\NotificationService;
use RuntimeException;

final class FinanceService
{
    public const DEFAULT_CATEGORIES = [
        'Maintenance and repairs',
        'Meals and entertainment',
        'Office expense',
        'Travel',
        'Motor vehicle expenses',
    ];

    public function __construct(
        private readonly FrankfurterService $frankfurter,
        private readonly NotificationService $notifications,
    ) {}

    public function seedDefaultCategories(Company $company): void
    {
        foreach (self::DEFAULT_CATEGORIES as $name) {
            ExpenseCategory::query()->firstOrCreate([
                'company_id' => $company->id,
                'name' => $name,
            ]);
        }
    }

    public function create(Company $company, Employee $employee, array $data): Expense
    {
        $currency = strtoupper($data['currency']);
        $conversion = [];
        if ($currency !== strtoupper($company->currency)) {
            $rate = $this->frankfurter->rate($data['expensed_at'], $currency, $company->currency);
            $conversion = [
                'converted_amount' => (int) round((int) $data['amount'] * $rate),
                'converted_to_currency' => strtoupper($company->currency),
                'converted_at' => now(),
                'exchange_rate' => $rate,
            ];
        }

        $managers = DirectReport::query()
            ->with('manager')
            ->where('company_id', $company->id)
            ->where('employee_id', $employee->id)
            ->get()
            ->pluck('manager')
            ->filter();

        return DB::transaction(function () use ($company, $employee, $data, $currency, $conversion, $managers) {
            $expense = Expense::query()->create([
                'company_id' => $company->id,
                'employee_id' => $employee->id,
                'expense_category_id' => $data['expense_category_id'] ?? null,
                'status' => $managers->isNotEmpty() ? 'manager_approval' : 'accounting_approval',
                'title' => $data['title'],
                'amount' => $data['amount'],
                'currency' => $currency,
                'description' => $data['description'] ?? null,
                'expensed_at' => $data['expensed_at'],
                ...$conversion,
            ]);

            foreach ($managers as $manager) {
                $this->notifications->create($company, $manager, 'expense.manager_approval_requested', [
                    'expense_id' => (string) $expense->id,
                    'employee_id' => (string) $employee->id,
                ]);
            }

            return $expense->load(['employee', 'category', 'managerApprover', 'accountingApprover']);
        });
    }

    public function find(Company $company, string $id): Expense
    {
        return Expense::query()
            ->with(['employee', 'category', 'managerApprover', 'accountingApprover'])
            ->where('company_id', $company->id)
            ->where('id', $id)
            ->firstOrFail();
    }

    public function visible(Company $company, Employee $actor, ?string $employeeId = null): Collection
    {
        $query = Expense::query()
            ->with(['employee', 'category', 'managerApprover', 'accountingApprover'])
            ->where('company_id', $company->id);

        if ($actor->hasAnyRole(['administrator', 'hr', 'accountant'])) {
            $query->when($employeeId, fn (Builder $q) => $q->where('employee_id', $employeeId));
        } else {
            $reportIds = DirectReport::query()
                ->where('company_id', $company->id)
                ->where('manager_id', $actor->id)
                ->pluck('employee_id');
            $query->where(function (Builder $q) use ($actor, $reportIds) {
                $q->where('employee_id', $actor->id)->orWhereIn('employee_id', $reportIds);
            });
            if ($employeeId) {
                $query->where('employee_id', $employeeId);
            }
        }

        return $query->orderByDesc('created_at')->get();
    }

    public function pendingManager(Company $company, Employee $manager): Collection
    {
        $reportIds = DirectReport::query()
            ->where('company_id', $company->id)
            ->where('manager_id', $manager->id)
            ->pluck('employee_id');

        return Expense::query()
            ->with(['employee', 'category', 'managerApprover', 'accountingApprover'])
            ->where('company_id', $company->id)
            ->where('status', 'manager_approval')
            ->whereIn('employee_id', $reportIds)
            ->orderBy('created_at')
            ->get();
    }

    public function pendingAccounting(Company $company): Collection
    {
        return Expense::query()
            ->with(['employee', 'category', 'managerApprover', 'accountingApprover'])
            ->where('company_id', $company->id)
            ->where('status', 'accounting_approval')
            ->orderBy('created_at')
            ->get();
    }

    public function managerDecision(Expense $expense, Employee $actor, bool $approve, ?string $reason = null): Expense
    {
        if ($expense->status !== 'manager_approval') {
            throw new RuntimeException('Expense is not awaiting manager approval', 409);
        }
        $isManager = DirectReport::query()
            ->where('company_id', $expense->company_id)
            ->where('manager_id', $actor->id)
            ->where('employee_id', $expense->employee_id)
            ->exists();
        if (! $isManager) {
            throw new RuntimeException('Forbidden', 403);
        }

        $expense->update([
            'status' => $approve ? 'accounting_approval' : 'rejected_by_manager',
            'manager_approver_id' => $actor->id,
            'manager_approver_approved_at' => $approve ? now() : null,
            'manager_rejection_explanation' => $approve ? null : $reason,
        ]);

        return $expense->fresh(['employee', 'category', 'managerApprover', 'accountingApprover']);
    }

    public function accountingDecision(Expense $expense, Employee $actor, bool $approve, ?string $reason = null): Expense
    {
        if ($expense->status !== 'accounting_approval') {
            throw new RuntimeException('Expense is not awaiting accounting approval', 409);
        }

        $expense->update([
            'status' => $approve ? 'accepted' : 'rejected_by_accounting',
            'accounting_approver_id' => $actor->id,
            'accounting_approver_approved_at' => $approve ? now() : null,
            'accounting_rejection_explanation' => $approve ? null : $reason,
        ]);

        return $expense->fresh(['employee', 'category', 'managerApprover', 'accountingApprover']);
    }

    public function delete(Expense $expense, Employee $actor): void
    {
        if ($expense->status === 'accepted') {
            throw new RuntimeException('Accepted expenses cannot be deleted', 409);
        }
        if ((string) $expense->employee_id !== (string) $actor->id && ! $actor->can('expenses.delete')) {
            throw new RuntimeException('Forbidden', 403);
        }
        $expense->delete();
    }

    public function payload(Expense $expense): array
    {
        $expense->loadMissing(['employee', 'category', 'managerApprover', 'accountingApprover']);

        return [
            'id' => (string) $expense->id,
            'company_id' => (string) $expense->company_id,
            'employee_id' => $expense->employee_id ? (string) $expense->employee_id : null,
            'employee_name' => $expense->employee?->fullName(),
            'expense_category_id' => $expense->expense_category_id ? (string) $expense->expense_category_id : null,
            'category' => $expense->category ? $this->categoryPayload($expense->category) : null,
            'status' => $expense->status,
            'title' => $expense->title,
            'amount' => (int) $expense->amount,
            'currency' => $expense->currency,
            'converted_amount' => $expense->converted_amount === null ? null : (int) $expense->converted_amount,
            'converted_to_currency' => $expense->converted_to_currency,
            'converted_at' => $expense->converted_at?->toIso8601String(),
            'exchange_rate' => $expense->exchange_rate === null ? null : (float) $expense->exchange_rate,
            'description' => $expense->description,
            'expensed_at' => $expense->expensed_at->toDateString(),
            'manager_approver_id' => $expense->manager_approver_id ? (string) $expense->manager_approver_id : null,
            'manager_approver_name' => $expense->managerApprover?->fullName(),
            'manager_approver_approved_at' => $expense->manager_approver_approved_at?->toIso8601String(),
            'manager_rejection_explanation' => $expense->manager_rejection_explanation,
            'accounting_approver_id' => $expense->accounting_approver_id ? (string) $expense->accounting_approver_id : null,
            'accounting_approver_name' => $expense->accountingApprover?->fullName(),
            'accounting_approver_approved_at' => $expense->accounting_approver_approved_at?->toIso8601String(),
            'accounting_rejection_explanation' => $expense->accounting_rejection_explanation,
        ];
    }

    public function categoryPayload(ExpenseCategory $category): array
    {
        return [
            'id' => (string) $category->id,
            'company_id' => (string) $category->company_id,
            'name' => $category->name,
        ];
    }
}
