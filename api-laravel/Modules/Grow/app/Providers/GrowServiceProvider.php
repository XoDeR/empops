<?php

namespace Modules\Grow\Providers;

use Modules\Grow\Console\ECoffeeStartCommand;
use Modules\Grow\Console\LogCompanyMoraleCommand;
use Modules\Grow\Console\LogTeamMoraleCommand;
use Modules\Grow\Console\RateManagerStartCommand;
use Modules\Grow\Console\RateManagerStopCommand;
use Nwidart\Modules\Support\ModuleServiceProvider;

class GrowServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Grow';

    protected string $nameLower = 'grow';

    protected array $providers = [
        RouteServiceProvider::class,
    ];

    public function boot(): void
    {
        parent::boot();

        if ($this->app->runningInConsole()) {
            $this->commands([
                LogCompanyMoraleCommand::class,
                LogTeamMoraleCommand::class,
                RateManagerStartCommand::class,
                RateManagerStopCommand::class,
                ECoffeeStartCommand::class,
            ]);
        }
    }
}
