<?php

namespace Modules\Employee\Services;

use App\Models\User;
use Illuminate\Support\Str;
use Modules\Company\Models\Company;
use Modules\Company\Services\FlowService;
use Modules\Employee\Models\DirectReport;
use Modules\Employee\Models\Employee;
use Modules\Time\Models\CompanyPtoPolicy;
use RuntimeException;

final class EmployeeService
{
    public function create(Company $company, array $data, string $role = 'employee'): Employee
    {
        $policy = CompanyPtoPolicy::query()
            ->where('company_id', $company->id)
            ->where('year', now()->year)
            ->first();
        $employee = Employee::query()->create([
            'company_id' => $company->id,
            'email' => $data['email'],
            'first_name' => $data['first_name'],
            'last_name' => $data['last_name'],
            'hired_at' => $data['hired_at'] ?? null,
            'position_id' => $data['position_id'] ?? null,
            'employee_status_id' => $data['employee_status_id'] ?? null,
            'locked' => false,
            'amount_of_allowed_holidays' => $policy?->default_amount_of_allowed_holidays,
            'amount_of_sick_days' => $policy?->default_amount_of_sick_days,
            'amount_of_pto_days' => $policy?->default_amount_of_pto_days,
        ]);

        $employee->assignRole($role);

        return $employee->load(['position', 'status', 'roles']);
    }

    public function update(Employee $employee, array $data): Employee
    {
        $wasLocked = (bool) $employee->locked;
        $employee->fill(collect($data)->only([
            'email',
            'first_name',
            'last_name',
            'hired_at',
            'position_id',
            'employee_status_id',
            'locked',
        ])->all());

        $employee->save();

        if (! $wasLocked && $employee->locked) {
            app(FlowService::class)->scheduleForEmployee($employee->company, $employee, 'employee_leaves_company', now());
        }

        if (isset($data['role'])) {
            $keepManager = $employee->hasRole('manager');
            $employee->syncRoles([$data['role']]);
            if ($keepManager) {
                $employee->assignRole('manager');
            }
        }

        return $employee->fresh(['position', 'status', 'roles']);
    }

    public function invite(Employee $employee): Employee
    {
        if ($employee->user_id !== null) {
            throw new RuntimeException('Employee already linked to a user', 409);
        }

        $employee->invitation_link = (string) Str::uuid();
        $employee->invitation_used_at = null;
        $employee->save();

        return $employee->fresh(['position', 'status', 'roles']);
    }

    public function acceptInvite(User $user, string $link): Employee
    {
        $employee = Employee::query()->where('invitation_link', $link)->first();

        if ($employee === null) {
            throw new RuntimeException('Invitation not found', 404);
        }

        if ($employee->invitation_used_at !== null || $employee->user_id !== null) {
            throw new RuntimeException('Invitation already used', 409);
        }

        if ($employee->locked) {
            throw new RuntimeException('Employee is locked', 403);
        }

        $alreadyMember = Employee::query()
            ->where('company_id', $employee->company_id)
            ->where('user_id', $user->id)
            ->where('id', '!=', $employee->id)
            ->exists();

        if ($alreadyMember) {
            throw new RuntimeException('Already a member of this company', 409);
        }

        $employee->user_id = $user->id;
        $employee->invitation_used_at = now();
        $employee->save();

        return $employee->fresh(['company', 'position', 'status', 'roles']);
    }

    /**
     * @return array{created: int, errors: list<array{row: int, message: string}>}
     */
    public function importFromCsv(Company $company, string $path): array
    {
        $handle = fopen($path, 'r');
        if ($handle === false) {
            throw new RuntimeException('Unable to read CSV file', 400);
        }

        $header = fgetcsv($handle);
        if ($header === false) {
            fclose($handle);
            throw new RuntimeException('CSV file is empty', 400);
        }

        $header = array_map(fn ($h) => strtolower(trim((string) $h)), $header);
        $required = ['email', 'first_name', 'last_name'];
        foreach ($required as $col) {
            if (! in_array($col, $header, true)) {
                fclose($handle);
                throw new RuntimeException("CSV missing required column: {$col}", 422);
            }
        }

        $created = 0;
        $errors = [];
        $rowNum = 1;

        while (($row = fgetcsv($handle)) !== false) {
            $rowNum++;
            if (count($row) === 1 && trim((string) $row[0]) === '') {
                continue;
            }

            $data = [];
            foreach ($header as $i => $key) {
                $data[$key] = isset($row[$i]) ? trim((string) $row[$i]) : '';
            }

            if ($data['email'] === '' || $data['first_name'] === '' || $data['last_name'] === '') {
                $errors[] = ['row' => $rowNum, 'message' => 'email, first_name, and last_name are required'];
                continue;
            }

            if (! filter_var($data['email'], FILTER_VALIDATE_EMAIL)) {
                $errors[] = ['row' => $rowNum, 'message' => 'invalid email'];
                continue;
            }

            if (Employee::query()->where('company_id', $company->id)->where('email', $data['email'])->exists()) {
                $errors[] = ['row' => $rowNum, 'message' => 'email already exists'];
                continue;
            }

            try {
                $this->create($company, [
                    'email' => $data['email'],
                    'first_name' => $data['first_name'],
                    'last_name' => $data['last_name'],
                    'hired_at' => $data['hired_at'] !== '' ? $data['hired_at'] : null,
                    'position_id' => $data['position_id'] !== '' ? $data['position_id'] : null,
                ], 'employee');
                $created++;
            } catch (\Throwable $e) {
                $errors[] = ['row' => $rowNum, 'message' => $e->getMessage()];
            }
        }

        fclose($handle);

        return ['created' => $created, 'errors' => $errors];
    }

    /**
     * @return array<string, mixed>
     */
    public function employeePayload(Employee $employee, bool $includeInvite = false): array
    {
        $employee->loadMissing([
            'position',
            'status',
            'roles',
            'teams',
            'managerLinks.manager',
            'managedReports',
            'media',
        ]);

        $managers = $employee->managerLinks
            ->map(fn (DirectReport $d) => [
                'id' => (string) $d->manager->id,
                'first_name' => $d->manager->first_name,
                'last_name' => $d->manager->last_name,
                'email' => $d->manager->email,
            ])
            ->values()
            ->all();

        $payload = [
            'id' => (string) $employee->id,
            'company_id' => (string) $employee->company_id,
            'user_id' => $employee->user_id ? (string) $employee->user_id : null,
            'email' => $employee->email,
            'first_name' => $employee->first_name,
            'last_name' => $employee->last_name,
            'hired_at' => $employee->hired_at?->toDateString(),
            'locked' => $employee->locked,
            'position' => $employee->position ? [
                'id' => (string) $employee->position->id,
                'title' => $employee->position->title,
            ] : null,
            'status' => $employee->status ? [
                'id' => (string) $employee->status->id,
                'name' => $employee->status->name,
                'type' => $employee->status->type,
            ] : null,
            'roles' => $employee->getRoleNames()->values()->all(),
            'manager' => $managers[0] ?? null,
            'managers' => $managers,
            'teams' => $employee->teams
                ->map(fn ($t) => [
                    'id' => (string) $t->id,
                    'name' => $t->name,
                ])
                ->values()
                ->all(),
            'is_manager' => $employee->hasRole('manager') || $employee->managedReports->isNotEmpty(),
            'avatar_url' => $employee->getFirstMediaUrl('avatar') ?: null,
        ];

        if ($includeInvite) {
            $payload['invitation_link'] = $employee->invitation_link;
            $payload['invitation_url'] = $employee->invitation_link
                ? '/invitations/'.$employee->invitation_link.'/accept'
                : null;
        }

        return $payload;
    }
}
