<?php

namespace Modules\Grow\Console;

use Carbon\Carbon;
use Illuminate\Console\Command;
use Modules\Grow\Services\GrowService;

class LogTeamMoraleCommand extends Command
{
    protected $signature = 'empops:log-team-morale {date? : YYYY-MM-DD (defaults to today)}';

    protected $description = 'Snapshot team morale averages for a date';

    public function handle(GrowService $grow): int
    {
        $date = $this->argument('date')
            ? Carbon::parse($this->argument('date'))
            : null;
        $count = $grow->logTeamMoraleForDate($date);
        $this->info("Created {$count} team morale history row(s).");

        return self::SUCCESS;
    }
}
