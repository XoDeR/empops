<?php

namespace Modules\Employee\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Position;

class PositionController extends Controller
{
    public function index(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        $items = Position::query()
            ->where('company_id', $company->id)
            ->orderBy('title')
            ->get()
            ->map(fn (Position $p) => [
                'id' => (string) $p->id,
                'company_id' => (string) $p->company_id,
                'title' => $p->title,
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
            'title' => ['required', 'string', 'max:255'],
        ]);

        $position = Position::query()->create([
            'company_id' => $company->id,
            'title' => $validated['title'],
        ]);

        return ApiResponse::success([
            'id' => (string) $position->id,
            'company_id' => (string) $position->company_id,
            'title' => $position->title,
        ], 'Position created', 201);
    }

    public function update(Request $request, string $companyId, string $positionId): JsonResponse
    {
        $position = $this->findInCompany($request, $positionId);

        $validated = $request->validate([
            'title' => ['required', 'string', 'max:255'],
        ]);

        $position->update($validated);

        return ApiResponse::success([
            'id' => (string) $position->id,
            'company_id' => (string) $position->company_id,
            'title' => $position->title,
        ], 'Position updated');
    }

    public function destroy(Request $request, string $companyId, string $positionId): JsonResponse
    {
        $this->findInCompany($request, $positionId)->delete();

        return ApiResponse::success(null, 'Position deleted');
    }

    private function findInCompany(Request $request, string $positionId): Position
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return Position::query()
            ->where('company_id', $company->id)
            ->where('id', $positionId)
            ->firstOrFail();
    }
}
