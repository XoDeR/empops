<?php

namespace Modules\Employee\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Employee\Models\Employee;
use Modules\Employee\Services\EmployeeService;
use Modules\Uploads\Services\MediaAttachService;
use RuntimeException;

class EmployeeAvatarController extends Controller
{
    public function __construct(
        private readonly EmployeeService $employees,
        private readonly MediaAttachService $mediaAttach,
    ) {}

    public function update(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $employee = Employee::query()
            ->where('company_id', $companyId)
            ->where('id', $employeeId)
            ->firstOrFail();

        $isSelf = (string) $actor->id === (string) $employee->id;
        if (! $isSelf && ! $actor->hasPermissionTo('employees.update')) {
            return ApiResponse::error('Forbidden', 403);
        }

        $validated = $request->validate([
            'temporary_upload_id' => ['required', 'integer', 'exists:temporary_uploads,id'],
            'media_id' => ['required', 'integer', 'exists:media,id'],
        ]);

        try {
            $this->mediaAttach->attachFromTemporary(
                $employee,
                'avatar',
                (int) $validated['temporary_upload_id'],
                (int) $validated['media_id'],
            );
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(
            $this->employees->employeePayload($employee->fresh()),
            'Avatar updated',
        );
    }
}
