<?php

namespace Modules\Place\Services\Geocoder;

interface Geocoder
{
    /**
     * @param  array{street?: ?string, city?: ?string, province?: ?string, postal_code?: ?string, country?: ?string}  $address
     * @return array{latitude: ?float, longitude: ?float}
     */
    public function geocode(array $address): array;
}
