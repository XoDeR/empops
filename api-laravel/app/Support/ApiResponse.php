<?php

namespace App\Support;

use Illuminate\Http\JsonResponse;

final class ApiResponse
{
    public static function success(mixed $data = null, string $message = 'OK', int $status = 200): JsonResponse
    {
        return response()->json([
            'success' => true,
            'message' => $message,
            'data' => $data,
            'error' => null,
            'timestamp' => now()->toIso8601String(),
        ], $status);
    }

    public static function error(string $message, int $status = 400, mixed $error = null): JsonResponse
    {
        return response()->json([
            'success' => false,
            'message' => $message,
            'data' => null,
            'error' => $error ?? ['message' => $message],
            'timestamp' => now()->toIso8601String(),
        ], $status);
    }
}
