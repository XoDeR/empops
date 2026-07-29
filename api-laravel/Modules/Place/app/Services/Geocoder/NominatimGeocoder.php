<?php

namespace Modules\Place\Services\Geocoder;

use Illuminate\Support\Facades\Http;
use RuntimeException;

final class NominatimGeocoder implements Geocoder
{
    public function __construct(
        private readonly string $baseUrl,
        private readonly string $userAgent,
    ) {}

    public function geocode(array $address): array
    {
        if (isset($address['latitude'], $address['longitude'])) {
            return [
                'latitude' => (float) $address['latitude'],
                'longitude' => (float) $address['longitude'],
            ];
        }

        $parts = array_filter([
            $address['street'] ?? null,
            $address['city'] ?? null,
            $address['province'] ?? null,
            $address['postal_code'] ?? null,
            $address['country'] ?? null,
        ]);

        if ($parts === []) {
            return ['latitude' => null, 'longitude' => null];
        }

        $response = Http::withHeaders(['User-Agent' => $this->userAgent])
            ->get($this->baseUrl, [
                'q' => implode(', ', $parts),
                'format' => 'json',
                'limit' => 1,
            ]);

        if (! $response->successful()) {
            throw new RuntimeException('Geocoding request failed', 502);
        }

        $results = $response->json();
        if (! is_array($results) || $results === []) {
            return ['latitude' => null, 'longitude' => null];
        }

        $first = $results[0];

        return [
            'latitude' => isset($first['lat']) ? (float) $first['lat'] : null,
            'longitude' => isset($first['lon']) ? (float) $first['lon'] : null,
        ];
    }
}
