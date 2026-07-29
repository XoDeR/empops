<?php

namespace Modules\Notification\Services;

use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Notification\Models\Notification;

final class NotificationService
{
    /**
     * @param  array<string, mixed>  $objects
     */
    public function create(Company $company, Employee $employee, string $action, array $objects = []): Notification
    {
        return Notification::query()->create([
            'company_id' => $company->id,
            'employee_id' => $employee->id,
            'action' => $action,
            'objects' => $objects,
        ]);
    }

    /**
     * @return array{items: list<array<string, mixed>>, unread_count: int}
     */
    public function listForEmployee(Company $company, Employee $employee): array
    {
        $items = Notification::query()
            ->where('company_id', $company->id)
            ->where('employee_id', $employee->id)
            ->orderByDesc('created_at')
            ->limit(100)
            ->get();

        $unread = $items->whereNull('read_at')->count();

        return [
            'items' => $items->map(fn (Notification $n) => $this->payload($n))->values()->all(),
            'unread_count' => $unread,
        ];
    }

    public function unreadCount(Company $company, Employee $employee): int
    {
        return Notification::query()
            ->where('company_id', $company->id)
            ->where('employee_id', $employee->id)
            ->whereNull('read_at')
            ->count();
    }

    /**
     * @param  list<string>|null  $ids
     */
    public function markRead(Company $company, Employee $employee, ?array $ids = null): void
    {
        $query = Notification::query()
            ->where('company_id', $company->id)
            ->where('employee_id', $employee->id)
            ->whereNull('read_at');

        if ($ids !== null && $ids !== []) {
            $query->whereIn('id', $ids);
        }

        $query->update(['read_at' => now()]);
    }

    /**
     * @return array<string, mixed>
     */
    public function payload(Notification $n): array
    {
        return [
            'id' => (string) $n->id,
            'company_id' => (string) $n->company_id,
            'employee_id' => (string) $n->employee_id,
            'action' => $n->action,
            'objects' => $n->objects ?? [],
            'read' => $n->read_at !== null,
            'read_at' => $n->read_at?->toIso8601String(),
            'created_at' => $n->created_at?->toIso8601String(),
            'updated_at' => $n->updated_at?->toIso8601String(),
        ];
    }
}
