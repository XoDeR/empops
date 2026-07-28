<?php

namespace Modules\Employee\Services;

use App\Models\User;
use Illuminate\Support\Str;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use RuntimeException;

final class EmployeeService
{
    public function create(Company $company, array $data, string $role = 'employee'): Employee
    {
        $employee = Employee::query()->create([
            'company_id' => $company->id,
            'email' => $data['email'],
            'first_name' => $data['first_name'],
            'last_name' => $data['last_name'],
            'hired_at' => $data['hired_at'] ?? null,
            'position_id' => $data['position_id'] ?? null,
            'employee_status_id' => $data['employee_status_id'] ?? null,
            'locked' => false,
        ]);

        $employee->assignRole($role);

        return $employee->load(['position', 'status', 'roles']);
    }

    public function update(Employee $employee, array $data): Employee
    {
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

        if (isset($data['role'])) {
            $employee->syncRoles([$data['role']]);
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
     * @return array<string, mixed>
     */
    public function employeePayload(Employee $employee, bool $includeInvite = false): array
    {
        $employee->loadMissing(['position', 'status', 'roles']);

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
