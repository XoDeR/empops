<?php

namespace Modules\Grow\Console;

use Illuminate\Console\Command;
use Modules\Grow\Services\GrowService;

class RateManagerStopCommand extends Command
{
    protected $signature = 'empops:rate-manager-stop {--force : Stop all active surveys}';

    protected $description = 'Stop expired rate-your-manager surveys';

    public function handle(GrowService $grow): int
    {
        $count = $grow->stopRateManagerSurveys((bool) $this->option('force'));
        $this->info("Stopped {$count} rate-your-manager survey(s).");

        return self::SUCCESS;
    }
}
