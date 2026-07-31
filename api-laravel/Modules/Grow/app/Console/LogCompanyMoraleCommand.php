<?php

namespace Modules\Grow\Console;

use Carbon\Carbon;
use Illuminate\Console\Command;
use Modules\Grow\Services\GrowService;

class LogCompanyMoraleCommand extends Command
{
    protected $signature = 'empops:log-company-morale {date? : YYYY-MM-DD (defaults to today)}';

    protected $description = 'Snapshot company morale averages for a date';

    public function handle(GrowService $grow): int
    {
        $date = $this->argument('date')
            ? Carbon::parse($this->argument('date'))
            : null;
        $count = $grow->logCompanyMoraleForDate($date);
        $this->info("Created {$count} company morale history row(s).");

        return self::SUCCESS;
    }
}
