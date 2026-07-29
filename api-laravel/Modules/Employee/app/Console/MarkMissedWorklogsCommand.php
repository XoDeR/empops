<?php

namespace Modules\Employee\Console;

use Illuminate\Console\Command;
use Modules\Employee\Services\WorklogService;

class MarkMissedWorklogsCommand extends Command
{
    protected $signature = 'empops:mark-missed-worklogs {date? : YYYY-MM-DD (defaults to today)}';

    protected $description = 'Increment consecutive_worklog_missed for employees without a worklog on the given date';

    public function handle(WorklogService $worklogs): int
    {
        $date = $this->argument('date') ?: now()->toDateString();
        $updated = $worklogs->markMissedForDate($date);
        $this->info("Marked missed worklogs for {$date}: {$updated} employees updated");

        return self::SUCCESS;
    }
}
