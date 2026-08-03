<?php

namespace Modules\Billing\Console;

use Illuminate\Console\Command;
use Modules\Billing\Services\BillingService;

class LogUsageCommand extends Command
{
    protected $signature = 'empops:log-usage {date?}';
    protected $description = 'Log daily active employee usage';

    public function handle(BillingService $billing): int
    {
        if (! config('empops.enable_paid_plan')) return self::SUCCESS;
        $count = $billing->logDailyUsage($this->argument('date'));
        $this->info("Logged usage for {$count} company(s).");

        return self::SUCCESS;
    }
}
