<?php

return [
    /*
    | Geocoder driver: noop (manual lat/lng only) or nominatim.
    */
    'geocoder' => env('GEOCODER_DRIVER', 'noop'),

    'nominatim' => [
        'url' => env('NOMINATIM_URL', 'https://nominatim.openstreetmap.org/search'),
        'user_agent' => env('NOMINATIM_USER_AGENT', 'EmpOps/1.0 (contact@empops.local)'),
    ],
];
