<?php

namespace Modules\Employee\Providers;

use Modules\Employee\Console\MarkMissedWorklogsCommand;
use Nwidart\Modules\Support\ModuleServiceProvider;

class EmployeeServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Employee';

    protected string $nameLower = 'employee';

    protected array $providers = [
        RouteServiceProvider::class,
    ];

    public function boot(): void
    {
        parent::boot();

        if ($this->app->runningInConsole()) {
            $this->commands([
                MarkMissedWorklogsCommand::class,
            ]);
        }
    }
}
