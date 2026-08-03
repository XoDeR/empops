<?php

namespace Modules\Time\Services;

use Carbon\Carbon;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\DB;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Time\Models\CompanyCalendar;
use Modules\Time\Models\CompanyPtoPolicy;
use Modules\Time\Models\EmployeeDailyCalendarEntry;
use Modules\Time\Models\EmployeePlannedHoliday;
use RuntimeException;

final class PtoService
{
    public function listPolicies(Company $company): Collection
    {
        return CompanyPtoPolicy::query()->where('company_id', $company->id)->orderByDesc('year')->get();
    }

    public function createPolicy(Company $company, array $data): CompanyPtoPolicy
    {
        return DB::transaction(function () use ($company, $data) {
            $policy = CompanyPtoPolicy::query()->create([
                'company_id' => $company->id,
                'year' => $data['year'],
                ...$this->defaults($data),
            ]);
            $day = Carbon::create((int) $policy->year, 1, 1)->startOfDay();
            $end = $day->copy()->endOfYear();
            $worked = 0;
            while ($day->lte($end)) {
                $isWorked = ! $day->isWeekend();
                $worked += $isWorked ? 1 : 0;
                CompanyCalendar::query()->create([
                    'company_pto_policy_id' => $policy->id,
                    'day' => $day->toDateString(),
                    'day_of_week' => $day->dayOfWeekIso,
                    'day_of_year' => $day->dayOfYear,
                    'is_worked' => $isWorked,
                ]);
                $day->addDay();
            }
            $policy->update(['total_worked_days' => $worked]);

            return $policy->fresh();
        });
    }

    public function updatePolicy(CompanyPtoPolicy $policy, array $data): CompanyPtoPolicy
    {
        $policy->fill($this->defaults($data, true))->save();

        return $policy->fresh();
    }

    public function getCalendar(CompanyPtoPolicy $policy): Collection
    {
        return $policy->calendar()->get();
    }

    public function toggleCalendarDay(CompanyPtoPolicy $policy, string $day, ?bool $isWorked = null): CompanyCalendar
    {
        $entry = $policy->calendar()->whereDate('day', $day)->first();
        if ($entry === null) {
            throw new RuntimeException('Calendar day not found', 404);
        }
        $entry->is_worked = $isWorked ?? ! $entry->is_worked;
        $entry->save();
        $policy->update(['total_worked_days' => $policy->calendar()->where('is_worked', true)->count()]);

        return $entry->fresh();
    }

    public function holidayBalance(Employee $employee): array
    {
        return [
            'employee_id' => $employee->id,
            'holiday_balance' => (float) $employee->holiday_balance,
            'amount_of_allowed_holidays' => $employee->amount_of_allowed_holidays === null ? null : (float) $employee->amount_of_allowed_holidays,
            'amount_of_sick_days' => $employee->amount_of_sick_days === null ? null : (float) $employee->amount_of_sick_days,
            'amount_of_pto_days' => $employee->amount_of_pto_days === null ? null : (float) $employee->amount_of_pto_days,
        ];
    }

    public function listHolidays(Employee $employee): Collection
    {
        return EmployeePlannedHoliday::query()->where('employee_id', $employee->id)->orderBy('planned_date')->get();
    }

    public function createHoliday(Employee $employee, array $data): EmployeePlannedHoliday
    {
        return EmployeePlannedHoliday::query()->create(['employee_id' => $employee->id, ...$data]);
    }

    public function deleteHoliday(EmployeePlannedHoliday $holiday): void
    {
        $holiday->delete();
    }

    public function calculateTimeOffForDate(string|Carbon $date): int
    {
        $date = $date instanceof Carbon ? $date->copy()->startOfDay() : Carbon::parse($date)->startOfDay();
        $count = 0;
        Employee::query()->with('company')->where('locked', false)->whereNotNull('company_id')->chunkById(100, function ($employees) use ($date, &$count) {
            foreach ($employees as $employee) {
                if (EmployeeDailyCalendarEntry::query()->where('employee_id', $employee->id)->whereDate('log_date', $date)->exists()) {
                    continue;
                }
                $policy = CompanyPtoPolicy::query()->where('company_id', $employee->company_id)->where('year', $date->year)->first();
                if ($policy === null || $policy->total_worked_days < 1) {
                    continue;
                }
                $worked = $policy->calendar()->whereDate('day', $date)->where('is_worked', true)->exists();
                if (! $worked) {
                    continue;
                }
                $allowed = (float) ($employee->amount_of_allowed_holidays ?? $policy->default_amount_of_allowed_holidays);
                $accrued = $allowed / $policy->total_worked_days;
                DB::transaction(function () use ($employee, $policy, $date, $allowed, $accrued) {
                    $employee->increment('holiday_balance', $accrued);
                    $employee->refresh();
                    EmployeeDailyCalendarEntry::query()->create([
                        'employee_id' => $employee->id,
                        'log_date' => $date->toDateString(),
                        'new_balance' => $employee->holiday_balance,
                        'daily_accrued_amount' => $accrued,
                        'current_holidays_per_year' => $allowed,
                        'default_amount_of_allowed_holidays_in_company' => $policy->default_amount_of_allowed_holidays,
                    ]);
                });
                $count++;
            }
        });

        return $count;
    }

    private function defaults(array $data, bool $partial = false): array
    {
        $keys = ['default_amount_of_allowed_holidays', 'default_amount_of_sick_days', 'default_amount_of_pto_days'];
        $result = [];
        foreach ($keys as $key) {
            if (array_key_exists($key, $data)) {
                $result[$key] = $data[$key];
            } elseif (! $partial) {
                $result[$key] = 0;
            }
        }

        return $result;
    }
}
