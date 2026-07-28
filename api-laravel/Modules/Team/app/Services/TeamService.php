<?php

namespace Modules\Team\Services;

use Illuminate\Support\Facades\DB;
use Modules\Company\Models\Company;
use Modules\Company\Services\AuditLogger;
use Modules\Employee\Models\Employee;
use Modules\Team\Models\Team;
use RuntimeException;

final class TeamService
{
    public function __construct(private readonly AuditLogger $audit) {}

    public function create(Company $company, array $data, ?Employee $actor = null): Team
    {
        $name = trim($data['name']);
        $this->assertUniqueName($company, $name);

        $team = Team::query()->create([
            'company_id' => $company->id,
            'name' => $name,
            'description' => $data['description'] ?? null,
        ]);

        $this->audit->log($company, $actor, 'team.created', $team, [
            'name' => $team->name,
        ]);

        return $team->load(['leader', 'employees']);
    }

    public function update(Team $team, array $data, ?Employee $actor = null): Team
    {
        if (isset($data['name'])) {
            $name = trim($data['name']);
            $this->assertUniqueName($team->company, $name, (string) $team->id);
            $team->name = $name;
        }

        if (array_key_exists('description', $data)) {
            $team->description = $data['description'];
        }

        $team->save();

        $this->audit->log($team->company, $actor, 'team.updated', $team, [
            'name' => $team->name,
        ]);

        return $team->fresh(['leader', 'employees']);
    }

    public function destroy(Team $team, ?Employee $actor = null): void
    {
        $company = $team->company;
        $payload = ['name' => $team->name, 'team_id' => (string) $team->id];
        $team->delete();

        $this->audit->log($company, $actor, 'team.deleted', null, $payload);
    }

    public function addMember(Team $team, Employee $employee, ?Employee $actor = null): Team
    {
        $this->assertSameCompany($team, $employee);

        if (! $team->employees()->where('employees.id', $employee->id)->exists()) {
            $team->employees()->attach($employee->id);
            $this->audit->log($team->company, $actor, 'team.member_added', $team, [
                'employee_id' => (string) $employee->id,
            ]);
        }

        return $team->fresh(['leader', 'employees']);
    }

    public function removeMember(Team $team, Employee $employee, ?Employee $actor = null): Team
    {
        $this->assertSameCompany($team, $employee);

        if ((string) $team->team_leader_id === (string) $employee->id) {
            $team->team_leader_id = null;
            $team->save();
        }

        $team->employees()->detach($employee->id);

        $this->audit->log($team->company, $actor, 'team.member_removed', $team, [
            'employee_id' => (string) $employee->id,
        ]);

        return $team->fresh(['leader', 'employees']);
    }

    public function setLead(Team $team, ?Employee $leader, ?Employee $actor = null): Team
    {
        return DB::transaction(function () use ($team, $leader, $actor) {
            if ($leader === null) {
                $team->team_leader_id = null;
                $team->save();
                $this->audit->log($team->company, $actor, 'team.lead_cleared', $team);

                return $team->fresh(['leader', 'employees']);
            }

            $this->assertSameCompany($team, $leader);

            if (! $team->employees()->where('employees.id', $leader->id)->exists()) {
                $team->employees()->attach($leader->id);
            }

            $team->team_leader_id = $leader->id;
            $team->save();

            $this->audit->log($team->company, $actor, 'team.lead_set', $team, [
                'employee_id' => (string) $leader->id,
            ]);

            return $team->fresh(['leader', 'employees']);
        });
    }

    /**
     * @return array<string, mixed>
     */
    public function teamPayload(Team $team): array
    {
        $team->loadMissing(['leader', 'employees']);

        return [
            'id' => (string) $team->id,
            'company_id' => (string) $team->company_id,
            'name' => $team->name,
            'description' => $team->description,
            'leader' => $team->leader ? $this->employeeSummary($team->leader) : null,
            'members' => $team->employees
                ->map(fn (Employee $e) => $this->employeeSummary($e))
                ->values()
                ->all(),
            'member_count' => $team->employees->count(),
        ];
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

    private function assertUniqueName(Company $company, string $name, ?string $exceptId = null): void
    {
        $exists = Team::query()
            ->where('company_id', $company->id)
            ->when($exceptId, fn ($q) => $q->where('id', '!=', $exceptId))
            ->whereRaw('LOWER(name) = ?', [mb_strtolower($name)])
            ->exists();

        if ($exists) {
            throw new RuntimeException('Team name already exists in this company', 422);
        }
    }

    private function assertSameCompany(Team $team, Employee $employee): void
    {
        if ((string) $team->company_id !== (string) $employee->company_id) {
            throw new RuntimeException('Employee must belong to the same company', 422);
        }
    }
}
