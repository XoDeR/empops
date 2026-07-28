<?php

namespace Modules\Team\Providers;

use Nwidart\Modules\Support\ModuleServiceProvider;

class TeamServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Team';

    protected string $nameLower = 'team';

    protected array $providers = [
        RouteServiceProvider::class,
    ];
}
