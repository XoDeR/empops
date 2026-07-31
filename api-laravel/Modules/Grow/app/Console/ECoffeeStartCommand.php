<?php

namespace Modules\Grow\Console;

use Illuminate\Console\Command;
use Modules\Grow\Services\GrowService;

class ECoffeeStartCommand extends Command
{
    protected $signature = 'empops:e-coffee-start';

    protected $description = 'Start e-coffee pairing sessions for companies with e_coffee_enabled';

    public function handle(GrowService $grow): int
    {
        $count = $grow->startECoffeeSessions();
        $this->info("Started {$count} e-coffee session(s).");

        return self::SUCCESS;
    }
}
