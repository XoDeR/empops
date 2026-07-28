<?php

namespace Modules\Company\Services;

use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Spatie\Activitylog\Models\Activity;

final class AuditLogger
{
    /**
     * @param  array<string, mixed>  $properties
     */
    public function log(
        Company $company,
        ?Employee $actor,
        string $event,
        mixed $subject = null,
        array $properties = [],
    ): Activity {
        $logger = activity()
            ->causedBy($actor)
            ->withProperties(array_merge($properties, [
                'company_id' => (string) $company->id,
                'event' => $event,
            ]))
            ->event($event);

        if ($subject !== null) {
            $logger->performedOn($subject);
        }

        return $logger->log($event);
    }

    /**
     * @return array{items: list<array<string, mixed>>, page: int, per_page: int, total: int}
     */
    public function listForCompany(Company $company, int $page = 1, int $perPage = 50): array
    {
        $query = Activity::query()
            ->where('properties->company_id', (string) $company->id)
            ->orderByDesc('created_at');

        $total = (clone $query)->count();
        $items = $query
            ->forPage($page, $perPage)
            ->get()
            ->map(fn (Activity $a) => $this->payload($a))
            ->values()
            ->all();

        return [
            'items' => $items,
            'page' => $page,
            'per_page' => $perPage,
            'total' => $total,
        ];
    }

    /**
     * @return array<string, mixed>
     */
    public function payload(Activity $activity): array
    {
        $props = $activity->properties?->toArray() ?? [];

        return [
            'id' => (string) $activity->id,
            'company_id' => $props['company_id'] ?? null,
            'event' => $activity->event ?? ($props['event'] ?? $activity->description),
            'description' => $activity->description,
            'subject_type' => $activity->subject_type,
            'subject_id' => $activity->subject_id ? (string) $activity->subject_id : null,
            'causer_type' => $activity->causer_type,
            'causer_id' => $activity->causer_id ? (string) $activity->causer_id : null,
            'properties' => $props,
            'created_at' => $activity->created_at?->toIso8601String(),
        ];
    }
}
