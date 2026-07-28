<?php

namespace Modules\Employee\Providers;

use Nwidart\Modules\Support\ModuleServiceProvider;

class EmployeeServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Employee';

    protected string $nameLower = 'employee';

    protected array $providers = [
        RouteServiceProvider::class,
    ];
}
