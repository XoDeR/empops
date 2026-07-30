<?php

namespace Modules\Time\Providers;

use Nwidart\Modules\Support\ModuleServiceProvider;

class TimeServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Time';

    protected string $nameLower = 'time';

    protected array $providers = [
        RouteServiceProvider::class,
    ];
}
