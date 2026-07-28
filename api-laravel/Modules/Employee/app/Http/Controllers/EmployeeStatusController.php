<?php

namespace Modules\Employee\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;
use Modules\Company\Models\Company;
use Modules\Employee\Models\EmployeeStatus;

class EmployeeStatusController extends Controller
{
    public function index(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        $items = EmployeeStatus::query()
            ->where('company_id', $company->id)
            ->orderBy('name')
            ->get()
            ->map(fn (EmployeeStatus $s) => [
                'id' => (string) $s->id,
                'company_id' => (string) $s->company_id,
                'name' => $s->name,
                'type' => $s->type,
            ])
            ->values()
            ->all();

        return ApiResponse::success($items);
    }

    public function store(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        $validated = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'type' => ['sometimes', Rule::in([EmployeeStatus::TYPE_INTERNAL, EmployeeStatus::TYPE_EXTERNAL])],
        ]);

        $status = EmployeeStatus::query()->create([
            'company_id' => $company->id,
            'name' => $validated['name'],
            'type' => $validated['type'] ?? EmployeeStatus::TYPE_INTERNAL,
        ]);

        return ApiResponse::success([
            'id' => (string) $status->id,
            'company_id' => (string) $status->company_id,
            'name' => $status->name,
            'type' => $status->type,
        ], 'Employee status created', 201);
    }

    public function update(Request $request, string $companyId, string $statusId): JsonResponse
    {
        $status = $this->findInCompany($request, $statusId);

        $validated = $request->validate([
            'name' => ['sometimes', 'string', 'max:255'],
            'type' => ['sometimes', Rule::in([EmployeeStatus::TYPE_INTERNAL, EmployeeStatus::TYPE_EXTERNAL])],
        ]);

        $status->update($validated);

        return ApiResponse::success([
            'id' => (string) $status->id,
            'company_id' => (string) $status->company_id,
            'name' => $status->name,
            'type' => $status->type,
        ], 'Employee status updated');
    }

    public function destroy(Request $request, string $companyId, string $statusId): JsonResponse
    {
        $this->findInCompany($request, $statusId)->delete();

        return ApiResponse::success(null, 'Employee status deleted');
    }

    private function findInCompany(Request $request, string $statusId): EmployeeStatus
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return EmployeeStatus::query()
            ->where('company_id', $company->id)
            ->where('id', $statusId)
            ->firstOrFail();
    }
}
