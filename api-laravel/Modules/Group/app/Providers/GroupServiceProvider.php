<?php

namespace Modules\Group\Providers;

use Nwidart\Modules\Support\ModuleServiceProvider;

class GroupServiceProvider extends ModuleServiceProvider
{
    protected string $name = 'Group';
    protected string $nameLower = 'group';
    protected array $providers = [RouteServiceProvider::class];
}
