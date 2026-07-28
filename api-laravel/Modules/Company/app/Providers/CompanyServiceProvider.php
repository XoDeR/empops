<?php

namespace Modules\Company\Providers;

use Nwidart\Modules\Support\ModuleServiceProvider;

class CompanyServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Company';

    protected string $nameLower = 'company';

    protected array $providers = [
        RouteServiceProvider::class,
    ];
}
