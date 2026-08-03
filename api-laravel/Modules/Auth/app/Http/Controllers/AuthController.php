<?php

namespace Modules\Auth\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Models\User;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rules\Password;
use Modules\Auth\Services\AuthService;
use RuntimeException;

class AuthController extends Controller
{
    public function __construct(private readonly AuthService $auth) {}

    public function register(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'email' => ['required', 'email', 'max:255', 'unique:users,email'],
            'password' => ['required', 'string', 'confirmed', Password::defaults()],
        ]);

        try {
            $payload = $this->auth->register(
                $validated['name'],
                $validated['email'],
                $validated['password'],
            );
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($payload, 'Registered', 201);
    }

    public function login(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'email' => ['required', 'email'],
            'password' => ['required', 'string'],
        ]);

        $payload = $this->auth->login($validated['email'], $validated['password']);

        return ApiResponse::success($payload, 'Logged in');
    }

    public function refresh(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'refresh_token' => ['required', 'string'],
        ]);

        try {
            $payload = $this->auth->refresh($validated['refresh_token']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), 401);
        }

        return ApiResponse::success($payload, 'Token refreshed');
    }

    public function logout(Request $request): JsonResponse
    {
        $refreshToken = $request->input('refresh_token');
        if (is_string($refreshToken)) {
            $this->auth->logout($refreshToken);
        }

        return ApiResponse::success(null, 'Logged out');
    }

    public function me(Request $request): JsonResponse
    {
        /** @var User $user */
        $user = $request->user();

        return ApiResponse::success($this->auth->userPayload($user), 'OK');
    }
}
