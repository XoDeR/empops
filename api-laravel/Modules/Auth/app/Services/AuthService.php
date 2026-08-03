<?php

namespace Modules\Auth\Services;

use App\Models\User;
use Illuminate\Support\Facades\Hash;
use Illuminate\Validation\ValidationException;
use Modules\Auth\Models\RefreshToken;
use RuntimeException;
use Throwable;

final class AuthService
{
    public function __construct(private readonly JwtService $jwt) {}

    /**
     * @return array{access_token: string, refresh_token: string, expires_in: int, token_type: string, user: array{id: string, email: string, name: string}}
     */
    public function register(string $name, string $email, string $password): array
    {
        if (! config('empops.enable_signups')) {
            throw new RuntimeException('Signups are disabled', 403);
        }

        $user = User::query()->create([
            'name' => $name,
            'email' => $email,
            'password' => $password,
        ]);

        return $this->issueAuthPayload($user);
    }

    /**
     * @return array{access_token: string, refresh_token: string, expires_in: int, token_type: string, user: array{id: string, email: string, name: string}}
     */
    public function login(string $email, string $password): array
    {
        $user = User::query()->where('email', $email)->first();

        if ($user === null || ! Hash::check($password, $user->password)) {
            throw ValidationException::withMessages([
                'email' => ['Invalid email or password.'],
            ]);
        }

        return $this->issueAuthPayload($user);
    }

    /**
     * @return array{access_token: string, refresh_token: string, expires_in: int, token_type: string, user: array{id: string, email: string, name: string}}
     */
    public function refresh(string $refreshToken): array
    {
        try {
            $claims = $this->jwt->decode($refreshToken);
        } catch (Throwable $e) {
            throw new RuntimeException('Invalid or expired refresh token', 401, $e);
        }

        if (($claims->type ?? null) !== 'refresh') {
            throw new RuntimeException('Refresh token required', 401);
        }

        $jti = (string) ($claims->jti ?? '');
        $userId = (string) ($claims->sub ?? '');

        $stored = RefreshToken::query()
            ->where('jti', $jti)
            ->where('user_id', $userId)
            ->first();

        if ($stored === null || ! $stored->isActive()) {
            throw new RuntimeException('Refresh token revoked or expired', 401);
        }

        $stored->update(['revoked_at' => now()]);

        $user = User::query()->find($userId);
        if ($user === null) {
            throw new RuntimeException('User not found', 401);
        }

        return $this->issueAuthPayload($user);
    }

    public function logout(?string $refreshToken): void
    {
        if ($refreshToken === null || $refreshToken === '') {
            return;
        }

        try {
            $claims = $this->jwt->decode($refreshToken);
        } catch (Throwable) {
            return;
        }

        if (($claims->type ?? null) !== 'refresh') {
            return;
        }

        RefreshToken::query()
            ->where('jti', (string) ($claims->jti ?? ''))
            ->whereNull('revoked_at')
            ->update(['revoked_at' => now()]);
    }

    /**
     * @return array{id: string, email: string, name: string}
     */
    public function userPayload(User $user): array
    {
        return [
            'id' => (string) $user->id,
            'email' => $user->email,
            'name' => $user->name,
        ];
    }

    /**
     * @return array{access_token: string, refresh_token: string, expires_in: int, token_type: string, user: array{id: string, email: string, name: string}}
     */
    private function issueAuthPayload(User $user): array
    {
        $tokens = $this->jwt->issueTokenPair((string) $user->id);
        $refreshClaims = $this->jwt->decode($tokens['refresh_token']);

        RefreshToken::query()->create([
            'user_id' => $user->id,
            'jti' => (string) $refreshClaims->jti,
            'expires_at' => now()->addSeconds((int) config('jwt.refresh_ttl')),
        ]);

        return [
            ...$tokens,
            'user' => $this->userPayload($user),
        ];
    }
}
