<?php

namespace Modules\Hardware\Providers;

use Nwidart\Modules\Support\ModuleServiceProvider;

class HardwareServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Hardware';

    protected string $nameLower = 'hardware';

    protected array $providers = [
        RouteServiceProvider::class,
    ];
}
