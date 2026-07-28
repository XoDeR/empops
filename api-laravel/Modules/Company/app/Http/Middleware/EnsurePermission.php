<?php

namespace Modules\Company\Http\Middleware;

use App\Support\ApiResponse;
use Closure;
use Illuminate\Http\Request;
use Modules\Employee\Models\Employee;
use Symfony\Component\HttpFoundation\Response;

final class EnsurePermission
{
    public function handle(Request $request, Closure $next, string ...$permissions): Response
    {
        /** @var Employee|null $employee */
        $employee = $request->attributes->get('employee');

        if ($employee === null) {
            return ApiResponse::error('Company membership required', 403);
        }

        foreach ($permissions as $permission) {
            if ($employee->hasPermissionTo($permission)) {
                return $next($request);
            }
        }

        return ApiResponse::error('Forbidden', 403);
    }
}
