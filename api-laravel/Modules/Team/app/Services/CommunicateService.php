<?php

namespace Modules\Team\Services;

use Illuminate\Support\Facades\DB;
use Modules\Company\Services\AuditLogger;
use Modules\Employee\Models\Employee;
use Modules\Notification\Services\NotificationService;
use Modules\Team\Models\Ship;
use Modules\Team\Models\Team;
use Modules\Team\Models\TeamNews;
use RuntimeException;

final class CommunicateService
{
    public function __construct(
        private readonly AuditLogger $audit,
        private readonly NotificationService $notifications,
    ) {}

    /**
     * @param  array{title: string, content: string}  $data
     */
    public function createTeamNews(Team $team, Employee $actor, array $data): TeamNews
    {
        $news = TeamNews::query()->create([
            'company_id' => $team->company_id,
            'team_id' => $team->id,
            'author_id' => $actor->id,
            'author_name' => $actor->fullName(),
            'title' => trim($data['title']),
            'content' => trim($data['content']),
        ]);

        $this->audit->log($team->company, $actor, 'team_news.created', $news, [
            'title' => $news->title,
            'team_id' => (string) $team->id,
        ]);

        return $news;
    }

    /**
     * @param  array{title?: string, content?: string}  $data
     */
    public function updateTeamNews(TeamNews $news, array $data, Employee $actor): TeamNews
    {
        if (isset($data['title'])) {
            $news->title = trim($data['title']);
        }
        if (isset($data['content'])) {
            $news->content = trim($data['content']);
        }
        $news->save();

        $this->audit->log($news->company, $actor, 'team_news.updated', $news);

        return $news->fresh();
    }

    public function destroyTeamNews(TeamNews $news, Employee $actor): void
    {
        $company = $news->company;
        $payload = ['news_id' => (string) $news->id, 'title' => $news->title];
        $news->delete();
        $this->audit->log($company, $actor, 'team_news.deleted', null, $payload);
    }

    /**
     * @param  array{title: string, description?: string|null, employee_ids?: list<string>}  $data
     */
    public function createShip(Team $team, Employee $actor, array $data): Ship
    {
        return DB::transaction(function () use ($team, $actor, $data) {
            $ship = Ship::query()->create([
                'company_id' => $team->company_id,
                'team_id' => $team->id,
                'author_id' => $actor->id,
                'author_name' => $actor->fullName(),
                'title' => trim($data['title']),
                'description' => isset($data['description']) ? trim((string) $data['description']) : null,
            ]);

            $employeeIds = array_values(array_unique($data['employee_ids'] ?? []));
            if ($employeeIds !== []) {
                $employees = Employee::query()
                    ->where('company_id', $team->company_id)
                    ->whereIn('id', $employeeIds)
                    ->get();

                if ($employees->count() !== count($employeeIds)) {
                    throw new RuntimeException('One or more employees not found in this company', 422);
                }

                $ship->employees()->attach($employees->pluck('id')->all());

                foreach ($employees as $employee) {
                    if ((string) $employee->id === (string) $actor->id) {
                        continue;
                    }
                    $this->notifications->create($team->company, $employee, 'employee_attached_to_recent_ship', [
                        'ship_title' => $ship->title,
                        'team_id' => (string) $team->id,
                        'ship_id' => (string) $ship->id,
                        'author_name' => $actor->fullName(),
                    ]);
                }
            }

            $this->audit->log($team->company, $actor, 'ship.created', $ship, [
                'title' => $ship->title,
                'team_id' => (string) $team->id,
            ]);

            return $ship->load('employees');
        });
    }

    public function destroyShip(Ship $ship, Employee $actor): void
    {
        $company = $ship->company;
        $payload = ['ship_id' => (string) $ship->id, 'title' => $ship->title];
        $ship->delete();
        $this->audit->log($company, $actor, 'ship.deleted', null, $payload);
    }

    /**
     * @return array<string, mixed>
     */
    public function teamNewsPayload(TeamNews $news): array
    {
        return [
            'id' => (string) $news->id,
            'company_id' => (string) $news->company_id,
            'team_id' => (string) $news->team_id,
            'author_id' => $news->author_id ? (string) $news->author_id : null,
            'author_name' => $news->author_name,
            'title' => $news->title,
            'content' => $news->content,
            'created_at' => $news->created_at?->toIso8601String(),
            'updated_at' => $news->updated_at?->toIso8601String(),
        ];
    }

    /**
     * @return array<string, mixed>
     */
    public function shipPayload(Ship $ship): array
    {
        $ship->loadMissing('employees');

        return [
            'id' => (string) $ship->id,
            'company_id' => (string) $ship->company_id,
            'team_id' => (string) $ship->team_id,
            'author_id' => $ship->author_id ? (string) $ship->author_id : null,
            'author_name' => $ship->author_name,
            'title' => $ship->title,
            'description' => $ship->description,
            'employees' => $ship->employees->map(fn (Employee $e) => [
                'id' => (string) $e->id,
                'first_name' => $e->first_name,
                'last_name' => $e->last_name,
                'email' => $e->email,
            ])->values()->all(),
            'created_at' => $ship->created_at?->toIso8601String(),
            'updated_at' => $ship->updated_at?->toIso8601String(),
        ];
    }

    public function canAccessTeam(Employee $actor, Team $team, string $permission): bool
    {
        if ($actor->hasPermissionTo($permission)) {
            return true;
        }

        return $team->employees()->where('employees.id', $actor->id)->exists();
    }

    public function canManageTeamNews(Employee $actor, TeamNews $news, string $permission): bool
    {
        if ((string) $actor->id === (string) $news->author_id) {
            return true;
        }

        return $actor->hasPermissionTo($permission);
    }

    public function canManageShip(Employee $actor, Ship $ship, string $permission): bool
    {
        if ((string) $actor->id === (string) $ship->author_id) {
            return true;
        }

        return $actor->hasPermissionTo($permission);
    }
}
