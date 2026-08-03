<?php

namespace Modules\Billing\Providers;

use Modules\Billing\Console\CreateInvoicesCommand;
use Modules\Billing\Console\LogUsageCommand;
use Nwidart\Modules\Support\ModuleServiceProvider;

class BillingServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Billing';
    protected string $nameLower = 'billing';
    protected array $providers = [];

    public function boot(): void
    {
        parent::boot();

        if (config('empops.enable_paid_plan')) {
            $this->app->register(RouteServiceProvider::class);
        }
        if ($this->app->runningInConsole()) {
            $this->commands([LogUsageCommand::class, CreateInvoicesCommand::class]);
        }
    }
}
