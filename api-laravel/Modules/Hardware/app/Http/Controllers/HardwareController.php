<?php

namespace Modules\Hardware\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Hardware\Models\Hardware;
use Modules\Hardware\Models\Software;
use Modules\Hardware\Services\HardwareService;
use RuntimeException;

class HardwareController extends Controller
{
    public function __construct(private readonly HardwareService $hardware) {}

    public function listHardware(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $status = $request->query('status', 'all');
        $q = $request->query('q');
        $items = $this->hardware->listHardware($company, is_string($status) ? $status : 'all', is_string($q) ? $q : null)
            ->map(fn (Hardware $h) => $this->hardware->hardwarePayload($h))
            ->values()
            ->all();

        return ApiResponse::success($items);
    }

    public function storeHardware(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $data = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'serial_number' => ['nullable', 'string', 'max:255'],
            'employee_id' => ['nullable', 'uuid'],
        ]);

        try {
            $item = $this->hardware->createHardware($company, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->hardware->hardwarePayload($item), 'Hardware created', 201);
    }

    public function showHardware(Request $request, string $companyId, string $hardwareId): JsonResponse
    {
        $item = $this->findHardware($request, $hardwareId);

        return ApiResponse::success($this->hardware->hardwarePayload($item));
    }

    public function updateHardware(Request $request, string $companyId, string $hardwareId): JsonResponse
    {
        $item = $this->findHardware($request, $hardwareId);
        $data = $request->validate([
            'name' => ['sometimes', 'required', 'string', 'max:255'],
            'serial_number' => ['nullable', 'string', 'max:255'],
        ]);
        $item = $this->hardware->updateHardware($item, $data);

        return ApiResponse::success($this->hardware->hardwarePayload($item), 'Hardware updated');
    }

    public function destroyHardware(Request $request, string $companyId, string $hardwareId): JsonResponse
    {
        $item = $this->findHardware($request, $hardwareId);
        $item->delete();

        return ApiResponse::success(null, 'Hardware deleted');
    }

    public function lendHardware(Request $request, string $companyId, string $hardwareId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $item = $this->findHardware($request, $hardwareId);
        $data = $request->validate(['employee_id' => ['required', 'uuid']]);

        try {
            $item = $this->hardware->lendHardware($company, $item, $data['employee_id']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->hardware->hardwarePayload($item), 'Hardware lent');
    }

    public function regainHardware(Request $request, string $companyId, string $hardwareId): JsonResponse
    {
        $item = $this->findHardware($request, $hardwareId);
        $item = $this->hardware->regainHardware($item);

        return ApiResponse::success($this->hardware->hardwarePayload($item), 'Hardware regained');
    }

    public function employeeHardware(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        if (! $this->canViewEmployeeAssets($request, $actor, $employeeId, 'hardware.view')) {
            return ApiResponse::error('Forbidden', 403);
        }

        $items = $this->hardware->employeeHardware($company, $employeeId)
            ->map(fn (Hardware $h) => $this->hardware->hardwarePayload($h))
            ->values()
            ->all();

        return ApiResponse::success($items);
    }

    public function listSoftwares(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $items = $this->hardware->listSoftwares($company)
            ->map(fn (Software $s) => $this->hardware->softwarePayload($s))
            ->values()
            ->all();

        return ApiResponse::success($items);
    }

    public function storeSoftware(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $data = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'product_key' => ['required', 'string'],
            'seats' => ['required', 'integer', 'min:1'],
            'website' => ['nullable', 'string', 'max:255'],
            'licensed_to_name' => ['nullable', 'string', 'max:255'],
            'licensed_to_email_address' => ['nullable', 'email', 'max:255'],
            'order_number' => ['nullable', 'string', 'max:255'],
            'purchase_amount' => ['nullable', 'integer', 'min:1'],
            'currency' => ['nullable', 'string', 'size:3'],
            'purchased_at' => ['nullable', 'date'],
        ]);

        $item = $this->hardware->createSoftware($company, $data);

        return ApiResponse::success($this->hardware->softwarePayload($item), 'Software created', 201);
    }

    public function showSoftware(Request $request, string $companyId, string $softwareId): JsonResponse
    {
        $item = $this->findSoftware($request, $softwareId);

        return ApiResponse::success($this->hardware->softwarePayload($item));
    }

    public function updateSoftware(Request $request, string $companyId, string $softwareId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $item = $this->findSoftware($request, $softwareId);
        $data = $request->validate([
            'name' => ['sometimes', 'required', 'string', 'max:255'],
            'product_key' => ['sometimes', 'required', 'string'],
            'seats' => ['sometimes', 'required', 'integer', 'min:1'],
            'website' => ['nullable', 'string', 'max:255'],
            'licensed_to_name' => ['nullable', 'string', 'max:255'],
            'licensed_to_email_address' => ['nullable', 'email', 'max:255'],
            'order_number' => ['nullable', 'string', 'max:255'],
            'purchase_amount' => ['nullable', 'integer', 'min:1'],
            'currency' => ['nullable', 'string', 'size:3'],
            'purchased_at' => ['nullable', 'date'],
        ]);
        $item = $this->hardware->updateSoftware($company, $item, $data);

        return ApiResponse::success($this->hardware->softwarePayload($item), 'Software updated');
    }

    public function destroySoftware(Request $request, string $companyId, string $softwareId): JsonResponse
    {
        $item = $this->findSoftware($request, $softwareId);
        $item->clearMediaCollection('software');
        $item->employees()->detach();
        $item->delete();

        return ApiResponse::success(null, 'Software deleted');
    }

    public function giveSeat(Request $request, string $companyId, string $softwareId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $item = $this->findSoftware($request, $softwareId);
        $data = $request->validate(['employee_id' => ['required', 'uuid']]);

        try {
            $item = $this->hardware->giveSeat($company, $item, $data['employee_id']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->hardware->softwarePayload($item), 'Seat given');
    }

    public function revokeSeat(Request $request, string $companyId, string $softwareId, string $employeeId): JsonResponse
    {
        $item = $this->findSoftware($request, $softwareId);
        $item = $this->hardware->revokeSeat($item, $employeeId);

        return ApiResponse::success($this->hardware->softwarePayload($item), 'Seat revoked');
    }

    public function giveSeatsToAll(Request $request, string $companyId, string $softwareId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $item = $this->findSoftware($request, $softwareId);
        $result = $this->hardware->giveSeatsToAll($company, $item);

        return ApiResponse::success([
            'assigned' => $result['assigned'],
            'software' => $this->hardware->softwarePayload($result['software']),
        ], 'Seats assigned');
    }

    public function employeesWithout(Request $request, string $companyId, string $softwareId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $item = $this->findSoftware($request, $softwareId);

        return ApiResponse::success($this->hardware->employeesWithout($company, $item));
    }

    public function attachFile(Request $request, string $companyId, string $softwareId): JsonResponse
    {
        $item = $this->findSoftware($request, $softwareId);
        $data = $request->validate([
            'temporary_upload_id' => ['required', 'integer', 'exists:temporary_uploads,id'],
            'media_id' => ['required', 'integer'],
        ]);
        $media = $this->hardware->attachFile($item, (int) $data['temporary_upload_id'], (int) $data['media_id']);

        return ApiResponse::success($this->hardware->mediaPayload($media), 'File attached');
    }

    public function detachFile(Request $request, string $companyId, string $softwareId, int $mediaId): JsonResponse
    {
        $item = $this->findSoftware($request, $softwareId);

        try {
            $this->hardware->detachFile($item, $mediaId);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'File detached');
    }

    public function employeeSoftwares(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        if (! $this->canViewEmployeeAssets($request, $actor, $employeeId, 'software.view')) {
            return ApiResponse::error('Forbidden', 403);
        }

        $items = $this->hardware->employeeSoftwares($company, $employeeId)
            ->map(fn (Software $s) => $this->hardware->softwarePayload($s))
            ->values()
            ->all();

        return ApiResponse::success($items);
    }

    private function findHardware(Request $request, string $hardwareId): Hardware
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Hardware $item */
        $item = Hardware::query()
            ->with('employee')
            ->where('company_id', $company->id)
            ->where('id', $hardwareId)
            ->firstOrFail();

        return $item;
    }

    private function findSoftware(Request $request, string $softwareId): Software
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Software $item */
        $item = Software::query()
            ->with(['employees', 'media'])
            ->where('company_id', $company->id)
            ->where('id', $softwareId)
            ->firstOrFail();

        return $item;
    }

    private function canViewEmployeeAssets(Request $request, Employee $actor, string $employeeId, string $viewPermission): bool
    {
        if ($actor->id === $employeeId) {
            return true;
        }

        return $actor->hasPermissionTo($viewPermission);
    }
}
