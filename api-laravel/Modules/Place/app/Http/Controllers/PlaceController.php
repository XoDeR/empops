<?php

namespace Modules\Place\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Place\Models\Place;
use Modules\Place\Services\PlaceService;
use RuntimeException;

class PlaceController extends Controller
{
    public function __construct(private readonly PlaceService $places) {}

    public function index(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $employee = $this->findEmployee($request, $employeeId);

        if (! $this->canViewPlaces($actor, $employee)) {
            return ApiResponse::error('Forbidden', 403);
        }

        $list = Place::query()
            ->with('country')
            ->where('placable_type', Employee::class)
            ->where('placable_id', $employee->id)
            ->orderByDesc('is_active')
            ->orderByDesc('created_at')
            ->get()
            ->map(fn (Place $p) => $this->places->placePayload($p))
            ->values()
            ->all();

        return ApiResponse::success($list);
    }

    public function store(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $employee = $this->findEmployee($request, $employeeId);

        if (! $this->canCreatePlace($actor, $employee)) {
            return ApiResponse::error('Forbidden', 403);
        }

        $validated = $request->validate([
            'street' => ['nullable', 'string', 'max:255'],
            'city' => ['nullable', 'string', 'max:255'],
            'province' => ['nullable', 'string', 'max:255'],
            'postal_code' => ['nullable', 'string', 'max:32'],
            'country_id' => ['nullable', 'uuid', 'exists:countries,id'],
            'latitude' => ['nullable', 'numeric', 'between:-90,90'],
            'longitude' => ['nullable', 'numeric', 'between:-180,180'],
            'is_active' => ['sometimes', 'boolean'],
        ]);

        try {
            $place = $this->places->createForEmployee($employee, $validated, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->places->placePayload($place), 'Place created', 201);
    }

    public function update(Request $request, string $companyId, string $placeId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $place = $this->findPlaceInCompany($request, $placeId);
        $employee = $place->placable;

        if (! $employee instanceof Employee || ! $this->canManagePlaces($actor, $employee)) {
            return ApiResponse::error('Forbidden', 403);
        }

        $validated = $request->validate([
            'street' => ['nullable', 'string', 'max:255'],
            'city' => ['nullable', 'string', 'max:255'],
            'province' => ['nullable', 'string', 'max:255'],
            'postal_code' => ['nullable', 'string', 'max:32'],
            'country_id' => ['nullable', 'uuid', 'exists:countries,id'],
            'latitude' => ['nullable', 'numeric', 'between:-90,90'],
            'longitude' => ['nullable', 'numeric', 'between:-180,180'],
            'is_active' => ['sometimes', 'boolean'],
        ]);

        try {
            $updated = $this->places->update($place, $validated, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->places->placePayload($updated), 'Place updated');
    }

    public function activate(Request $request, string $companyId, string $placeId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $place = $this->findPlaceInCompany($request, $placeId);
        $employee = $place->placable;

        if (! $employee instanceof Employee || ! $this->canManagePlaces($actor, $employee)) {
            return ApiResponse::error('Forbidden', 403);
        }

        $updated = $this->places->activate($place, $actor);

        return ApiResponse::success($this->places->placePayload($updated), 'Place activated');
    }

    public function destroy(Request $request, string $companyId, string $placeId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $place = $this->findPlaceInCompany($request, $placeId);
        $employee = $place->placable;

        if (! $employee instanceof Employee || ! $this->canManagePlaces($actor, $employee)) {
            return ApiResponse::error('Forbidden', 403);
        }

        $this->places->delete($place, $actor);

        return ApiResponse::success(null, 'Place deleted');
    }

    private function canViewPlaces(Employee $actor, Employee $subject): bool
    {
        if ((string) $actor->id === (string) $subject->id) {
            return true;
        }

        return $actor->hasPermissionTo('places.view');
    }

    private function canCreatePlace(Employee $actor, Employee $subject): bool
    {
        if ((string) $actor->id === (string) $subject->id) {
            return true;
        }

        return $actor->hasPermissionTo('places.create');
    }

    private function canManagePlaces(Employee $actor, Employee $subject): bool
    {
        if ((string) $actor->id === (string) $subject->id) {
            return true;
        }

        return $actor->hasPermissionTo('places.update');
    }

    private function findEmployee(Request $request, string $employeeId): Employee
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return Employee::query()
            ->where('company_id', $company->id)
            ->where('id', $employeeId)
            ->firstOrFail();
    }

    private function findPlaceInCompany(Request $request, string $placeId): Place
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return Place::query()
            ->with('placable')
            ->where('id', $placeId)
            ->where('placable_type', Employee::class)
            ->whereHasMorph('placable', [Employee::class], fn ($q) => $q->where('company_id', $company->id))
            ->firstOrFail();
    }
}
