<?php

namespace Modules\Finance\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\DirectReport;
use Modules\Employee\Models\Employee;
use Modules\Finance\Models\ExpenseCategory;
use Modules\Finance\Services\FinanceService;
use RuntimeException;

class FinanceController extends Controller
{
    public function __construct(private readonly FinanceService $finance) {}

    public function categories(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $items = ExpenseCategory::query()
            ->where('company_id', $company->id)
            ->orderBy('name')
            ->get()
            ->map(fn (ExpenseCategory $category) => $this->finance->categoryPayload($category))
            ->values()
            ->all();

        return ApiResponse::success($items);
    }

    public function createCategory(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $data = $request->validate(['name' => ['required', 'string', 'max:255']]);
        $category = ExpenseCategory::query()->create(['company_id' => $company->id, 'name' => $data['name']]);

        return ApiResponse::success($this->finance->categoryPayload($category), 'Expense category created', 201);
    }

    public function updateCategory(Request $request, string $companyId, string $categoryId): JsonResponse
    {
        $category = $this->category($request, $categoryId);
        $data = $request->validate(['name' => ['required', 'string', 'max:255']]);
        $category->update($data);

        return ApiResponse::success($this->finance->categoryPayload($category), 'Expense category updated');
    }

    public function deleteCategory(Request $request, string $companyId, string $categoryId): JsonResponse
    {
        $this->category($request, $categoryId)->delete();

        return ApiResponse::success(null, 'Expense category deleted');
    }

    public function expenses(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $data = $request->validate(['employeeId' => ['nullable', 'uuid']]);

        return ApiResponse::success(
            $this->finance->visible($company, $actor, $data['employeeId'] ?? null)
                ->map(fn ($expense) => $this->finance->payload($expense))
                ->values()
                ->all(),
        );
    }

    public function createExpense(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $data = $request->validate([
            'title' => ['required', 'string', 'max:255'],
            'amount' => ['required', 'integer', 'min:1'],
            'currency' => ['required', 'string', 'size:3'],
            'expensed_at' => ['required', 'date'],
            'expense_category_id' => ['nullable', 'uuid'],
            'description' => ['nullable', 'string'],
        ]);
        if (! empty($data['expense_category_id'])) {
            $this->category($request, $data['expense_category_id']);
        }

        try {
            $expense = $this->finance->create($company, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->finance->payload($expense), 'Expense created', 201);
    }

    public function pendingManager(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        return ApiResponse::success(
            $this->finance->pendingManager($company, $actor)
                ->map(fn ($expense) => $this->finance->payload($expense))
                ->values()
                ->all(),
        );
    }

    public function pendingAccounting(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return ApiResponse::success(
            $this->finance->pendingAccounting($company)
                ->map(fn ($expense) => $this->finance->payload($expense))
                ->values()
                ->all(),
        );
    }

    public function showExpense(Request $request, string $companyId, string $expenseId): JsonResponse
    {
        $expense = $this->finance->find($request->attributes->get('company'), $expenseId);
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        if (! $this->canView($actor, $expense->employee_id)) {
            return ApiResponse::error('Forbidden', 403);
        }

        return ApiResponse::success($this->finance->payload($expense));
    }

    public function deleteExpense(Request $request, string $companyId, string $expenseId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $expense = $this->finance->find($request->attributes->get('company'), $expenseId);

        try {
            $this->finance->delete($expense, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Expense deleted');
    }

    public function managerApprove(Request $request, string $companyId, string $expenseId): JsonResponse
    {
        return $this->managerDecision($request, $expenseId, true);
    }

    public function managerReject(Request $request, string $companyId, string $expenseId): JsonResponse
    {
        $request->validate(['reason' => ['required', 'string']]);

        return $this->managerDecision($request, $expenseId, false, $request->string('reason')->toString());
    }

    public function accountingApprove(Request $request, string $companyId, string $expenseId): JsonResponse
    {
        return $this->accountingDecision($request, $expenseId, true);
    }

    public function accountingReject(Request $request, string $companyId, string $expenseId): JsonResponse
    {
        $request->validate(['reason' => ['required', 'string']]);

        return $this->accountingDecision($request, $expenseId, false, $request->string('reason')->toString());
    }

    public function grantAccountant(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        if ($forbidden = $this->ensureHr($request)) {
            return $forbidden;
        }
        $employee = $this->employee($request, $employeeId);
        if (! $employee->hasRole('accountant')) {
            $employee->assignRole('accountant');
        }

        return ApiResponse::success(null, 'Accountant granted');
    }

    public function revokeAccountant(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        if ($forbidden = $this->ensureHr($request)) {
            return $forbidden;
        }
        $employee = $this->employee($request, $employeeId);
        if ($employee->hasRole('accountant')) {
            $employee->removeRole('accountant');
        }

        return ApiResponse::success(null, 'Accountant revoked');
    }

    private function managerDecision(Request $request, string $expenseId, bool $approve, ?string $reason = null): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $expense = $this->finance->find($request->attributes->get('company'), $expenseId);

        try {
            $expense = $this->finance->managerDecision($expense, $actor, $approve, $reason);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->finance->payload($expense));
    }

    private function accountingDecision(Request $request, string $expenseId, bool $approve, ?string $reason = null): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $expense = $this->finance->find($request->attributes->get('company'), $expenseId);

        try {
            $expense = $this->finance->accountingDecision($expense, $actor, $approve, $reason);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->finance->payload($expense));
    }

    private function canView(Employee $actor, ?string $employeeId): bool
    {
        if ($actor->hasAnyRole(['administrator', 'hr', 'accountant'])
            || (string) $actor->id === (string) $employeeId) {
            return true;
        }

        return DirectReport::query()
            ->where('company_id', $actor->company_id)
            ->where('manager_id', $actor->id)
            ->where('employee_id', $employeeId)
            ->exists();
    }

    private function ensureHr(Request $request): ?JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        return $actor->hasAnyRole(['administrator', 'hr']) ? null : ApiResponse::error('Forbidden', 403);
    }

    private function category(Request $request, string $id): ExpenseCategory
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return ExpenseCategory::query()->where('company_id', $company->id)->where('id', $id)->firstOrFail();
    }

    private function employee(Request $request, string $id): Employee
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return Employee::query()->where('company_id', $company->id)->where('id', $id)->firstOrFail();
    }
}
