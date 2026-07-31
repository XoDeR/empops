<?php

namespace Modules\Grow\Console;

use Illuminate\Console\Command;
use Modules\Grow\Services\GrowService;

class RateManagerStartCommand extends Command
{
    protected $signature = 'empops:rate-manager-start';

    protected $description = 'Start rate-your-manager surveys for all managers with direct reports';

    public function handle(GrowService $grow): int
    {
        $count = $grow->startRateManagerSurveys();
        $this->info("Started {$count} rate-your-manager survey(s).");

        return self::SUCCESS;
    }
}
