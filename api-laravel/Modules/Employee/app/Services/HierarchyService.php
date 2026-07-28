<?php

namespace Modules\Employee\Services;

use Illuminate\Support\Facades\DB;
use Modules\Company\Models\Company;
use Modules\Company\Services\AuditLogger;
use Modules\Employee\Models\DirectReport;
use Modules\Employee\Models\Employee;
use RuntimeException;

final class HierarchyService
{
    public function __construct(private readonly AuditLogger $audit) {}

    public function assignManager(Company $company, Employee $employee, Employee $manager, ?Employee $actor = null): DirectReport
    {
        if ((string) $employee->id === (string) $manager->id) {
            throw new RuntimeException('Cannot assign employee as their own manager', 422);
        }

        if ((string) $employee->company_id !== (string) $company->id
            || (string) $manager->company_id !== (string) $company->id) {
            throw new RuntimeException('Employees must belong to the company', 422);
        }

        $existing = DirectReport::query()
            ->where('manager_id', $manager->id)
            ->where('employee_id', $employee->id)
            ->first();

        if ($existing !== null) {
            return $existing->load(['manager', 'employee']);
        }

        return DB::transaction(function () use ($company, $employee, $manager, $actor) {
            $edge = DirectReport::query()->create([
                'company_id' => $company->id,
                'manager_id' => $manager->id,
                'employee_id' => $employee->id,
            ]);

            $this->syncManagerRole($manager);

            $this->audit->log($company, $actor, 'hierarchy.manager_assigned', $employee, [
                'manager_id' => (string) $manager->id,
                'employee_id' => (string) $employee->id,
            ]);

            return $edge->load(['manager', 'employee']);
        });
    }

    public function unassignManager(Company $company, Employee $employee, Employee $manager, ?Employee $actor = null): void
    {
        $edge = DirectReport::query()
            ->where('company_id', $company->id)
            ->where('manager_id', $manager->id)
            ->where('employee_id', $employee->id)
            ->first();

        if ($edge === null) {
            throw new RuntimeException('Manager relationship not found', 404);
        }

        DB::transaction(function () use ($company, $employee, $manager, $actor, $edge) {
            $edge->delete();
            $this->syncManagerRole($manager);

            $this->audit->log($company, $actor, 'hierarchy.manager_unassigned', $employee, [
                'manager_id' => (string) $manager->id,
                'employee_id' => (string) $employee->id,
            ]);
        });
    }

    public function syncManagerRole(Employee $manager): void
    {
        $hasReports = DirectReport::query()
            ->where('manager_id', $manager->id)
            ->exists();

        if ($hasReports) {
            if (! $manager->hasRole('manager')) {
                $manager->assignRole('manager');
            }
        } elseif ($manager->hasRole('manager')) {
            $manager->removeRole('manager');
        }
    }

    /**
     * @return array{id: string, first_name: string, last_name: string, email: string}
     */
    public function employeeSummary(Employee $employee): array
    {
        return [
            'id' => (string) $employee->id,
            'first_name' => $employee->first_name,
            'last_name' => $employee->last_name,
            'email' => $employee->email,
        ];
    }
}
