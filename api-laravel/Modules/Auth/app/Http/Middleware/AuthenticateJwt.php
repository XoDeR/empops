<?php

namespace Modules\Auth\Http\Middleware;

use App\Support\ApiResponse;
use Closure;
use Illuminate\Http\Request;
use Modules\Auth\Services\JwtService;
use Symfony\Component\HttpFoundation\Response;
use Throwable;

final class AuthenticateJwt
{
    public function __construct(private readonly JwtService $jwt) {}

    public function handle(Request $request, Closure $next): Response
    {
        $header = $request->header('Authorization', '');

        if (! preg_match('/^Bearer\s+(.+)$/i', $header, $matches)) {
            return ApiResponse::error('Missing or invalid Authorization header', 401);
        }

        try {
            $claims = $this->jwt->decode($matches[1]);
        } catch (Throwable $e) {
            return ApiResponse::error('Invalid or expired token', 401, ['detail' => $e->getMessage()]);
        }

        if (($claims->type ?? null) !== 'access') {
            return ApiResponse::error('Access token required', 401);
        }

        $request->attributes->set('jwt_sub', $claims->sub ?? null);
        $request->attributes->set('jwt_claims', $claims);

        return $next($request);
    }
}
