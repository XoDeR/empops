<?php

namespace Modules\Finance\Providers;

use Nwidart\Modules\Support\ModuleServiceProvider;

class FinanceServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Finance';

    protected string $nameLower = 'finance';

    protected array $providers = [
        RouteServiceProvider::class,
    ];
}
