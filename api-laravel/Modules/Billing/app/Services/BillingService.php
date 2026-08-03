<?php

namespace Modules\Billing\Services;

use Carbon\Carbon;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\DB;
use Modules\Billing\Models\CompanyDailyUsageHistory;
use Modules\Billing\Models\CompanyInvoice;
use Modules\Billing\Models\CompanyUsageHistoryDetail;
use Modules\Company\Models\Company;

final class BillingService
{
    public function listInvoices(Company $company): Collection
    {
        return CompanyInvoice::query()->with('usage.details')->where('company_id', $company->id)->latest()->get();
    }

    public function logDailyUsage(string|Carbon|null $date = null): int
    {
        $date = $date instanceof Carbon ? $date : Carbon::parse($date ?: now());
        $count = 0;
        Company::query()->with(['employees' => fn ($q) => $q->where('locked', false)])->chunkById(100, function ($companies) use ($date, &$count) {
            foreach ($companies as $company) {
                DB::transaction(function () use ($company, $date) {
                    $usage = CompanyDailyUsageHistory::query()->updateOrCreate(
                        ['company_id' => $company->id, 'logged_on' => $date->toDateString()],
                        ['number_of_active_employees' => $company->employees->count()],
                    );
                    $usage->details()->delete();
                    foreach ($company->employees as $employee) {
                        CompanyUsageHistoryDetail::query()->create([
                            'usage_history_id' => $usage->id,
                            'employee_id' => $employee->id,
                            'employee_name' => $employee->fullName(),
                            'employee_email' => $employee->email,
                        ]);
                    }
                });
                $count++;
            }
        });

        return $count;
    }

    public function createMonthlyInvoices(string|Carbon|null $month = null): int
    {
        $month = $month instanceof Carbon ? $month : Carbon::parse($month ?: 'first day of last month');
        $start = $month->copy()->startOfMonth();
        $end = $month->copy()->endOfMonth();
        $count = 0;
        Company::query()->chunkById(100, function ($companies) use ($start, $end, &$count) {
            foreach ($companies as $company) {
                $usage = CompanyDailyUsageHistory::query()
                    ->where('company_id', $company->id)
                    ->whereBetween('logged_on', [$start->toDateString(), $end->toDateString()])
                    ->orderByDesc('number_of_active_employees')
                    ->orderByDesc('logged_on')
                    ->first();
                $usage ??= CompanyDailyUsageHistory::query()
                    ->where('company_id', $company->id)
                    ->whereDate('logged_on', '<=', $end)
                    ->latest('logged_on')
                    ->first();
                if ($usage === null) continue;
                $invoice = CompanyInvoice::query()->firstOrCreate([
                    'company_id' => $company->id,
                    'usage_history_id' => $usage->id,
                ]);
                if ($invoice->wasRecentlyCreated) $count++;
            }
        });

        return $count;
    }
}
