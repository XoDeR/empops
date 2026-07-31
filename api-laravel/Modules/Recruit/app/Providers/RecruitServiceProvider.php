<?php

namespace Modules\Recruit\Providers;

use Nwidart\Modules\Support\ModuleServiceProvider;

class RecruitServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Recruit';

    protected string $nameLower = 'recruit';

    protected array $providers = [
        RouteServiceProvider::class,
    ];
}
