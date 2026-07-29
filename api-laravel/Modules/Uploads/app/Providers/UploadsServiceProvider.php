<?php

namespace Modules\Uploads\Providers;

use Nwidart\Modules\Support\ModuleServiceProvider;

class UploadsServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Uploads';
    protected string $nameLower = 'uploads';

    protected array $providers = [
        RouteServiceProvider::class,
    ];
}

