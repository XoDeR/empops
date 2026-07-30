<?php

namespace Modules\Finance\Services;

use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Http;
use RuntimeException;

final class FrankfurterService
{
    public function rate(string $date, string $from, string $to): float
    {
        $from = strtoupper($from);
        $to = strtoupper($to);
        $key = "frankfurter:{$date}:{$from}:{$to}";

        return (float) Cache::remember($key, now()->addDay(), function () use ($date, $from, $to) {
            $response = Http::acceptJson()
                ->timeout(10)
                ->get("https://api.frankfurter.app/{$date}", [
                    'from' => $from,
                    'to' => $to,
                ]);

            if (! $response->successful() || ! is_numeric($response->json("rates.{$to}"))) {
                throw new RuntimeException('Unable to retrieve exchange rate', 502);
            }

            return (float) $response->json("rates.{$to}");
        });
    }
}
