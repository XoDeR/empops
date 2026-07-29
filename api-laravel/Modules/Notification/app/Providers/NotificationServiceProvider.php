<?php

namespace Modules\Notification\Providers;

use Nwidart\Modules\Support\ModuleServiceProvider;

class NotificationServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Notification';

    protected string $nameLower = 'notification';

    protected array $providers = [
        RouteServiceProvider::class,
    ];
}
