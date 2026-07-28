<?php

namespace Modules\Company\Http\Middleware;

use App\Support\ApiResponse;
use Closure;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Symfony\Component\HttpFoundation\Response;

final class EnsureCompanyMember
{
    public function handle(Request $request, Closure $next): Response
    {
        $companyId = (string) $request->route('companyId');
        $user = $request->user();

        if ($user === null) {
            return ApiResponse::error('Unauthenticated', 401);
        }

        $company = Company::query()->find($companyId);
        if ($company === null) {
            return ApiResponse::error('Company not found', 404);
        }

        $employee = Employee::query()
            ->where('company_id', $company->id)
            ->where('user_id', $user->id)
            ->first();

        if ($employee === null || $employee->locked) {
            return ApiResponse::error('Not a member of this company', 403);
        }

        $request->attributes->set('company', $company);
        $request->attributes->set('employee', $employee);

        return $next($request);
    }
}
