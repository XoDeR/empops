<?php

namespace Modules\Time\Providers;

use Modules\Time\Console\CalculateTimeOffCommand;
use Nwidart\Modules\Support\ModuleServiceProvider;

class TimeServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Time';

    protected string $nameLower = 'time';

    protected array $providers = [
        RouteServiceProvider::class,
    ];

    public function boot(): void
    {
        parent::boot();

        if ($this->app->runningInConsole()) {
            $this->commands([CalculateTimeOffCommand::class]);
        }
    }
}
