<?php

namespace Modules\Auth\Services;

use Firebase\JWT\JWT;
use Firebase\JWT\Key;
use Illuminate\Support\Str;
use stdClass;

final class JwtService
{
    public function issueAccessToken(string $userId, array $extra = []): string
    {
        return $this->encode($userId, 'access', (int) config('jwt.access_ttl'), $extra);
    }

    public function issueRefreshToken(string $userId): string
    {
        return $this->encode($userId, 'refresh', (int) config('jwt.refresh_ttl'));
    }

    public function decode(string $token): stdClass
    {
        return JWT::decode($token, new Key((string) config('jwt.secret'), 'HS256'));
    }

    /**
     * @return array{access_token: string, refresh_token: string, expires_in: int, token_type: string}
     */
    public function issueTokenPair(string $userId): array
    {
        return [
            'access_token' => $this->issueAccessToken($userId),
            'refresh_token' => $this->issueRefreshToken($userId),
            'expires_in' => (int) config('jwt.access_ttl'),
            'token_type' => 'Bearer',
        ];
    }

    private function encode(string $userId, string $type, int $ttl, array $extra = []): string
    {
        $now = time();

        $payload = array_merge([
            'sub' => $userId,
            'jti' => (string) Str::uuid(),
            'iat' => $now,
            'exp' => $now + $ttl,
            'iss' => (string) config('jwt.issuer'),
            'aud' => (string) config('jwt.audience'),
            'type' => $type,
        ], $extra);

        return JWT::encode($payload, (string) config('jwt.secret'), 'HS256');
    }
}
