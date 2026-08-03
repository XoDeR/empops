<?php

use Illuminate\Foundation\Inspiring;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Schedule;

Artisan::command('inspire', function () {
    $this->comment(Inspiring::quote());
})->purpose('Display an inspiring quote');

Schedule::command('empops:mark-missed-worklogs')->dailyAt('23:00');
Schedule::command('empops:log-company-morale')->dailyAt('23:00');
Schedule::command('empops:log-team-morale')->dailyAt('23:00');
Schedule::command('empops:rate-manager-start')->dailyAt('01:00')->when(fn () => now()->isLastOfMonth());
Schedule::command('empops:rate-manager-stop')->hourly();
Schedule::command('empops:e-coffee-start')->weeklyOn(1, '09:00');
Schedule::command('empops:calculate-timeoff')->dailyAt('23:00');
Schedule::command('empops:process-flows')->dailyAt('23:05');
Schedule::command('empops:log-usage')->dailyAt('23:10')->when(fn () => (bool) config('empops.enable_paid_plan'));
Schedule::command('empops:create-invoices')->monthlyOn(1, '00:15')->when(fn () => (bool) config('empops.enable_paid_plan'));
