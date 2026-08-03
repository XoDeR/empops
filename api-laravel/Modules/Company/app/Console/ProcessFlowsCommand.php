<?php

namespace Modules\Company\Console;

use Illuminate\Console\Command;
use Modules\Company\Services\FlowService;

class ProcessFlowsCommand extends Command
{
    protected $signature = 'empops:process-flows {date?}';
    protected $description = 'Process due employee flow actions';

    public function handle(FlowService $flows): int
    {
        $count = $flows->processDueRuns($this->argument('date'));
        $this->info("Processed {$count} flow action run(s).");

        return self::SUCCESS;
    }
}
