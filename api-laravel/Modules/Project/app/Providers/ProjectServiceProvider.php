<?php

namespace Modules\Project\Providers;

use Nwidart\Modules\Support\ModuleServiceProvider;

class ProjectServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Project';

    protected string $nameLower = 'project';

    protected array $providers = [
        RouteServiceProvider::class,
    ];
}
