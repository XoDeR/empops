<?php

namespace Modules\Billing\Console;

use Illuminate\Console\Command;
use Modules\Billing\Services\BillingService;

class CreateInvoicesCommand extends Command
{
    protected $signature = 'empops:create-invoices {month?}';
    protected $description = 'Create monthly company invoices';

    public function handle(BillingService $billing): int
    {
        if (! config('empops.enable_paid_plan')) return self::SUCCESS;
        $count = $billing->createMonthlyInvoices($this->argument('month'));
        $this->info("Created {$count} invoice(s).");

        return self::SUCCESS;
    }
}
