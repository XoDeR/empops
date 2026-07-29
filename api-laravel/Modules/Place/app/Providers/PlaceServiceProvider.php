<?php

namespace Modules\Place\Providers;

use Modules\Place\Services\Geocoder\Geocoder;
use Modules\Place\Services\Geocoder\NominatimGeocoder;
use Modules\Place\Services\Geocoder\NoopGeocoder;
use Nwidart\Modules\Support\ModuleServiceProvider;

class PlaceServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Place';

    protected string $nameLower = 'place';

    protected array $providers = [
        RouteServiceProvider::class,
    ];

    public function register(): void
    {
        parent::register();

        $this->app->singleton(Geocoder::class, function () {
            $driver = config('place.geocoder', 'noop');

            return match ($driver) {
                'nominatim' => new NominatimGeocoder(
                    config('place.nominatim.url', 'https://nominatim.openstreetmap.org/search'),
                    config('place.nominatim.user_agent', 'EmpOps/1.0'),
                ),
                default => new NoopGeocoder(),
            };
        });
    }
}
