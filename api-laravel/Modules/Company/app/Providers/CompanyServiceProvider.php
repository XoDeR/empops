<?php

namespace Modules\Company\Providers;

use Modules\Company\Console\ProcessFlowsCommand;
use Nwidart\Modules\Support\ModuleServiceProvider;

class CompanyServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Company';

    protected string $nameLower = 'company';

    protected array $providers = [
        RouteServiceProvider::class,
    ];

    public function boot(): void
    {
        parent::boot();

        if ($this->app->runningInConsole()) {
            $this->commands([ProcessFlowsCommand::class]);
        }
    }
}
