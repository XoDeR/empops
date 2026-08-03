<?php

namespace Modules\Time\Console;

use Illuminate\Console\Command;
use Modules\Time\Services\PtoService;

class CalculateTimeOffCommand extends Command
{
    protected $signature = 'empops:calculate-timeoff {date?}';
    protected $description = 'Accrue employee holiday balances for a date';

    public function handle(PtoService $pto): int
    {
        $date = $this->argument('date') ?: now()->toDateString();
        $count = $pto->calculateTimeOffForDate($date);
        $this->info("Accrued time off for {$count} employee(s).");

        return self::SUCCESS;
    }
}
