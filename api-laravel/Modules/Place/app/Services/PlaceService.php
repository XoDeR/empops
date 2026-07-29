<?php

namespace Modules\Place\Services;

use Illuminate\Support\Facades\DB;
use Modules\Company\Services\AuditLogger;
use Modules\Employee\Models\Employee;
use Modules\Place\Models\Country;
use Modules\Place\Models\Place;
use Modules\Place\Services\Geocoder\Geocoder;
use RuntimeException;

final class PlaceService
{
    public function __construct(
        private readonly Geocoder $geocoder,
        private readonly AuditLogger $audit,
    ) {}

    public function createForEmployee(Employee $employee, array $data, ?Employee $actor = null): Place
    {
        return DB::transaction(function () use ($employee, $data, $actor) {
            $isActive = (bool) ($data['is_active'] ?? false);

            if ($isActive) {
                Place::query()
                    ->where('placable_type', Employee::class)
                    ->where('placable_id', $employee->id)
                    ->update(['is_active' => false]);
            }

            $coords = $this->resolveCoordinates($data);

            $place = Place::query()->create([
                'placable_type' => Employee::class,
                'placable_id' => $employee->id,
                'street' => $data['street'] ?? null,
                'city' => $data['city'] ?? null,
                'province' => $data['province'] ?? null,
                'postal_code' => $data['postal_code'] ?? null,
                'country_id' => $data['country_id'] ?? null,
                'latitude' => $coords['latitude'],
                'longitude' => $coords['longitude'],
                'is_active' => $isActive,
            ]);

            if ($actor !== null) {
                $employee->loadMissing('company');
                $this->audit->log($employee->company, $actor, 'place.created', $place, [
                    'employee_id' => (string) $employee->id,
                ]);
            }

            return $place->load('country');
        });
    }

    public function update(Place $place, array $data, ?Employee $actor = null): Place
    {
        return DB::transaction(function () use ($place, $data, $actor) {
            $place->fill(collect($data)->only([
                'street',
                'city',
                'province',
                'postal_code',
                'country_id',
            ])->all());

            if (array_key_exists('latitude', $data) || array_key_exists('longitude', $data)) {
                $place->latitude = $data['latitude'] ?? null;
                $place->longitude = $data['longitude'] ?? null;
            } elseif ($this->addressFieldsChanged($place, $data)) {
                $coords = $this->resolveCoordinates(array_merge($place->toArray(), $data));
                $place->latitude = $coords['latitude'];
                $place->longitude = $coords['longitude'];
            }

            if (array_key_exists('is_active', $data) && $data['is_active']) {
                Place::query()
                    ->where('placable_type', $place->placable_type)
                    ->where('placable_id', $place->placable_id)
                    ->where('id', '!=', $place->id)
                    ->update(['is_active' => false]);
                $place->is_active = true;
            } elseif (array_key_exists('is_active', $data)) {
                $place->is_active = (bool) $data['is_active'];
            }

            $place->save();

            if ($actor !== null && $place->placable instanceof Employee) {
                $place->placable->loadMissing('company');
                $this->audit->log($place->placable->company, $actor, 'place.updated', $place);
            }

            return $place->fresh(['country']);
        });
    }

    public function activate(Place $place, ?Employee $actor = null): Place
    {
        return $this->update($place, ['is_active' => true], $actor);
    }

    public function delete(Place $place, ?Employee $actor = null): void
    {
        DB::transaction(function () use ($place, $actor) {
            $employee = $place->placable;
            $placeId = (string) $place->id;
            $place->delete();

            if ($actor !== null && $employee instanceof Employee) {
                $employee->loadMissing('company');
                $this->audit->log($employee->company, $actor, 'place.deleted', null, [
                    'place_id' => $placeId,
                ]);
            }
        });
    }

    /**
     * @return array<string, mixed>
     */
    public function placePayload(Place $place): array
    {
        $place->loadMissing('country');

        return [
            'id' => (string) $place->id,
            'street' => $place->street,
            'city' => $place->city,
            'province' => $place->province,
            'postal_code' => $place->postal_code,
            'country' => $place->country ? [
                'id' => (string) $place->country->id,
                'name' => $place->country->name,
                'code' => $place->country->code,
            ] : null,
            'latitude' => $place->latitude,
            'longitude' => $place->longitude,
            'is_active' => $place->is_active,
        ];
    }

    /**
     * @return array{latitude: ?float, longitude: ?float}
     */
    private function resolveCoordinates(array $data): array
    {
        if (array_key_exists('latitude', $data) || array_key_exists('longitude', $data)) {
            return [
                'latitude' => isset($data['latitude']) ? (float) $data['latitude'] : null,
                'longitude' => isset($data['longitude']) ? (float) $data['longitude'] : null,
            ];
        }

        $countryName = null;
        if (! empty($data['country_id'])) {
            $countryName = Country::query()->where('id', $data['country_id'])->value('name');
        }

        return $this->geocoder->geocode([
            'street' => $data['street'] ?? null,
            'city' => $data['city'] ?? null,
            'province' => $data['province'] ?? null,
            'postal_code' => $data['postal_code'] ?? null,
            'country' => $countryName,
        ]);
    }

    private function addressFieldsChanged(Place $place, array $data): bool
    {
        foreach (['street', 'city', 'province', 'postal_code', 'country_id'] as $field) {
            if (array_key_exists($field, $data) && $data[$field] !== $place->{$field}) {
                return true;
            }
        }

        return false;
    }
}
