<?php

namespace Modules\Place\Services\Geocoder;

final class NoopGeocoder implements Geocoder
{
    public function geocode(array $address): array
    {
        return [
            'latitude' => isset($address['latitude']) ? (float) $address['latitude'] : null,
            'longitude' => isset($address['longitude']) ? (float) $address['longitude'] : null,
        ];
    }
}
