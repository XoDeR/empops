<?php

namespace Modules\Auth\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Auth\Services\JwtService;
use Throwable;

class AuthController extends Controller
{
    /** Step 0 stub user — replaced by real users/RBAC in Step 1. */
    private const STUB_USER = [
        'id' => '00000000-0000-7000-8000-000000000001',
        'email' => 'dev@empops.local',
        'name' => 'EmpOps Dev',
    ];

    public function __construct(private readonly JwtService $jwt) {}

    public function login(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'email' => ['required', 'email'],
            'password' => ['required', 'string'],
        ]);

        $user = [
            ...self::STUB_USER,
            'email' => $validated['email'],
        ];

        $tokens = $this->jwt->issueTokenPair($user['id']);

        return ApiResponse::success([
            ...$tokens,
            'user' => $user,
        ], 'Logged in (stub)');
    }

    public function refresh(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'refresh_token' => ['required', 'string'],
        ]);

        try {
            $claims = $this->jwt->decode($validated['refresh_token']);
        } catch (Throwable $e) {
            return ApiResponse::error('Invalid or expired refresh token', 401, ['detail' => $e->getMessage()]);
        }

        if (($claims->type ?? null) !== 'refresh') {
            return ApiResponse::error('Refresh token required', 401);
        }

        $userId = (string) ($claims->sub ?? self::STUB_USER['id']);
        $tokens = $this->jwt->issueTokenPair($userId);

        return ApiResponse::success($tokens, 'Token refreshed (stub)');
    }

    public function logout(): JsonResponse
    {
        return ApiResponse::success(null, 'Logged out (stub)');
    }

    public function me(Request $request): JsonResponse
    {
        $sub = $request->attributes->get('jwt_sub', self::STUB_USER['id']);

        return ApiResponse::success([
            ...self::STUB_USER,
            'id' => $sub,
        ], 'Current user (stub)');
    }
}
