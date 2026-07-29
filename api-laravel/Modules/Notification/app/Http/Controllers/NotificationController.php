<?php

namespace Modules\Notification\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Notification\Services\NotificationService;

class NotificationController extends Controller
{
    public function __construct(private readonly NotificationService $notifications) {}

    public function index(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        return ApiResponse::success($this->notifications->listForEmployee($company, $actor));
    }

    public function markRead(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $validated = $request->validate([
            'ids' => ['nullable', 'array'],
            'ids.*' => ['uuid'],
        ]);

        $this->notifications->markRead($company, $actor, $validated['ids'] ?? null);

        return ApiResponse::success(null, 'Notifications marked as read');
    }
}
