<?php

namespace Modules\Hardware\Services;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Support\Collection;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Finance\Services\FrankfurterService;
use Modules\Hardware\Models\Hardware;
use Modules\Hardware\Models\Software;
use Modules\Uploads\Services\MediaAttachService;
use RuntimeException;
use Spatie\MediaLibrary\MediaCollections\Models\Media;
use Throwable;

final class HardwareService
{
    public function __construct(
        private readonly FrankfurterService $frankfurter,
        private readonly MediaAttachService $mediaAttach,
    ) {}

    public function listHardware(Company $company, ?string $status = 'all', ?string $q = null): Collection
    {
        $query = Hardware::query()
            ->with('employee')
            ->where('company_id', $company->id)
            ->orderBy('name');

        if ($status === 'available') {
            $query->whereNull('employee_id');
        } elseif ($status === 'lent') {
            $query->whereNotNull('employee_id');
        }

        if ($q !== null && $q !== '') {
            $like = '%'.$q.'%';
            $query->where(function (Builder $b) use ($like) {
                $b->where('name', 'ilike', $like)
                    ->orWhere('serial_number', 'ilike', $like);
            });
        }

        return $query->get();
    }

    public function createHardware(Company $company, array $data): Hardware
    {
        $employeeId = $data['employee_id'] ?? null;
        if ($employeeId) {
            $this->assertEmployeeInCompany($company, $employeeId);
        }

        return Hardware::query()->create([
            'company_id' => $company->id,
            'name' => $data['name'],
            'serial_number' => $data['serial_number'] ?? null,
            'employee_id' => $employeeId,
        ])->load('employee');
    }

    public function updateHardware(Hardware $hardware, array $data): Hardware
    {
        if (isset($data['name'])) {
            $hardware->name = $data['name'];
        }
        if (array_key_exists('serial_number', $data)) {
            $hardware->serial_number = $data['serial_number'];
        }
        $hardware->save();

        return $hardware->fresh('employee');
    }

    public function lendHardware(Company $company, Hardware $hardware, string $employeeId): Hardware
    {
        $this->assertEmployeeInCompany($company, $employeeId);
        $hardware->update(['employee_id' => $employeeId]);

        return $hardware->fresh('employee');
    }

    public function regainHardware(Hardware $hardware): Hardware
    {
        $hardware->update(['employee_id' => null]);

        return $hardware->fresh('employee');
    }

    public function employeeHardware(Company $company, string $employeeId): Collection
    {
        return Hardware::query()
            ->with('employee')
            ->where('company_id', $company->id)
            ->where('employee_id', $employeeId)
            ->orderBy('name')
            ->get();
    }

    public function listSoftwares(Company $company): Collection
    {
        return Software::query()
            ->with(['employees', 'media'])
            ->where('company_id', $company->id)
            ->orderBy('name')
            ->get();
    }

    public function createSoftware(Company $company, array $data): Software
    {
        $attrs = $this->softwareAttrs($data);
        $attrs['company_id'] = $company->id;
        $attrs = array_merge($attrs, $this->convertPurchase($company, $attrs));

        return Software::query()->create($attrs)->load(['employees', 'media']);
    }

    public function updateSoftware(Company $company, Software $software, array $data): Software
    {
        $attrs = $this->softwareAttrs($data, partial: true);
        foreach ($attrs as $key => $value) {
            $software->{$key} = $value;
        }

        if (array_key_exists('purchase_amount', $attrs) || array_key_exists('currency', $attrs) || array_key_exists('purchased_at', $attrs)) {
            $conversion = $this->convertPurchase($company, [
                'purchase_amount' => $software->purchase_amount,
                'currency' => $software->currency,
                'purchased_at' => $software->purchased_at?->format('Y-m-d'),
            ]);
            foreach ($conversion as $key => $value) {
                $software->{$key} = $value;
            }
            if ($software->purchase_amount === null || $software->currency === null) {
                $software->converted_purchase_amount = null;
                $software->converted_to_currency = null;
                $software->converted_at = null;
                $software->exchange_rate = null;
            }
        }

        $software->save();

        return $software->fresh(['employees', 'media']);
    }

    public function giveSeat(Company $company, Software $software, string $employeeId): Software
    {
        $this->assertEmployeeInCompany($company, $employeeId);
        $used = $software->employees()->count();
        if ($used >= $software->seats) {
            throw new RuntimeException('No seats remaining', 422);
        }
        if ($software->employees()->where('employees.id', $employeeId)->exists()) {
            return $software->fresh(['employees', 'media']);
        }
        $software->employees()->attach($employeeId);

        return $software->fresh(['employees', 'media']);
    }

    public function revokeSeat(Software $software, string $employeeId): Software
    {
        $software->employees()->detach($employeeId);

        return $software->fresh(['employees', 'media']);
    }

    /** @return array{assigned: int, software: Software} */
    public function giveSeatsToAll(Company $company, Software $software): array
    {
        $used = $software->employees()->count();
        $remaining = max(0, $software->seats - $used);
        if ($remaining === 0) {
            return ['assigned' => 0, 'software' => $software->fresh(['employees', 'media'])];
        }

        $assignedIds = $software->employees()->pluck('employees.id')->all();
        $eligible = Employee::query()
            ->where('company_id', $company->id)
            ->where('locked', false)
            ->when(count($assignedIds) > 0, fn (Builder $q) => $q->whereNotIn('id', $assignedIds))
            ->orderBy('first_name')
            ->limit($remaining)
            ->pluck('id')
            ->all();

        if ($eligible !== []) {
            $software->employees()->attach($eligible);
        }

        return [
            'assigned' => count($eligible),
            'software' => $software->fresh(['employees', 'media']),
        ];
    }

    /** @return array{employees_without: int, remaining_seats: int, seats: int} */
    public function employeesWithout(Company $company, Software $software): array
    {
        $assignedIds = $software->employees()->pluck('employees.id')->all();
        $without = Employee::query()
            ->where('company_id', $company->id)
            ->where('locked', false)
            ->when(count($assignedIds) > 0, fn (Builder $q) => $q->whereNotIn('id', $assignedIds))
            ->count();
        $used = count($assignedIds);

        return [
            'employees_without' => $without,
            'remaining_seats' => max(0, $software->seats - $used),
            'seats' => $software->seats,
        ];
    }

    public function employeeSoftwares(Company $company, string $employeeId): Collection
    {
        return Software::query()
            ->with(['employees', 'media'])
            ->where('company_id', $company->id)
            ->whereHas('employees', fn (Builder $q) => $q->where('employees.id', $employeeId))
            ->orderBy('name')
            ->get();
    }

    public function attachFile(Software $software, int $temporaryUploadId, int $mediaId): Media
    {
        return $this->mediaAttach->attachFromTemporary(
            $software,
            'software',
            $temporaryUploadId,
            $mediaId,
            clearExisting: false,
        );
    }

    public function detachFile(Software $software, int $mediaId): void
    {
        $media = $software->getMedia('software')->firstWhere('id', $mediaId);
        if ($media === null) {
            throw new RuntimeException('File not found', 404);
        }
        $media->delete();
    }

    public function hardwarePayload(Hardware $hardware): array
    {
        $employee = $hardware->employee;

        return [
            'id' => $hardware->id,
            'company_id' => $hardware->company_id,
            'name' => $hardware->name,
            'serial_number' => $hardware->serial_number,
            'employee_id' => $hardware->employee_id,
            'employee' => $employee ? [
                'id' => $employee->id,
                'first_name' => $employee->first_name,
                'last_name' => $employee->last_name,
                'email' => $employee->email,
            ] : null,
            'created_at' => $hardware->created_at?->toIso8601String(),
            'updated_at' => $hardware->updated_at?->toIso8601String(),
        ];
    }

    public function softwarePayload(Software $software, bool $includeRelations = true): array
    {
        $used = $includeRelations
            ? $software->employees->count()
            : $software->employees()->count();

        $payload = [
            'id' => $software->id,
            'company_id' => $software->company_id,
            'name' => $software->name,
            'product_key' => $software->product_key,
            'seats' => $software->seats,
            'seats_used' => $used,
            'remaining_seats' => max(0, $software->seats - $used),
            'website' => $software->website,
            'licensed_to_name' => $software->licensed_to_name,
            'licensed_to_email_address' => $software->licensed_to_email_address,
            'order_number' => $software->order_number,
            'purchase_amount' => $software->purchase_amount,
            'currency' => $software->currency,
            'converted_purchase_amount' => $software->converted_purchase_amount,
            'converted_to_currency' => $software->converted_to_currency,
            'converted_at' => $software->converted_at?->toIso8601String(),
            'exchange_rate' => $software->exchange_rate === null ? null : (float) $software->exchange_rate,
            'purchased_at' => $software->purchased_at?->format('Y-m-d'),
            'created_at' => $software->created_at?->toIso8601String(),
            'updated_at' => $software->updated_at?->toIso8601String(),
        ];

        if ($includeRelations) {
            $payload['employees'] = $software->employees->map(fn (Employee $e) => [
                'id' => $e->id,
                'first_name' => $e->first_name,
                'last_name' => $e->last_name,
                'email' => $e->email,
            ])->values()->all();

            $payload['files'] = $software->getMedia('software')->map(fn (Media $m) => [
                'id' => $m->id,
                'file_name' => $m->file_name,
                'mime_type' => $m->mime_type,
                'size' => $m->size,
                'url' => url('/api/v1/media/'.$m->id.'/file'),
            ])->values()->all();
        }

        return $payload;
    }

    public function mediaPayload(Media $media): array
    {
        return [
            'id' => $media->id,
            'file_name' => $media->file_name,
            'mime_type' => $media->mime_type,
            'size' => $media->size,
            'url' => url('/api/v1/media/'.$media->id.'/file'),
        ];
    }

    private function softwareAttrs(array $data, bool $partial = false): array
    {
        $keys = [
            'name', 'product_key', 'seats', 'website', 'licensed_to_name',
            'licensed_to_email_address', 'order_number', 'purchase_amount',
            'currency', 'purchased_at',
        ];
        $attrs = [];
        foreach ($keys as $key) {
            if (! array_key_exists($key, $data)) {
                if (! $partial && in_array($key, ['name', 'seats', 'product_key'], true)) {
                    continue;
                }
                continue;
            }
            $value = $data[$key];
            if ($key === 'currency' && $value !== null) {
                $value = strtoupper((string) $value);
            }
            $attrs[$key] = $value;
        }

        return $attrs;
    }

    /** @return array<string, mixed> */
    private function convertPurchase(Company $company, array $attrs): array
    {
        $amount = $attrs['purchase_amount'] ?? null;
        $currency = isset($attrs['currency']) ? strtoupper((string) $attrs['currency']) : null;
        if ($amount === null || $currency === null || $currency === '') {
            return [
                'converted_purchase_amount' => null,
                'converted_to_currency' => null,
                'converted_at' => null,
                'exchange_rate' => null,
            ];
        }

        $companyCurrency = strtoupper($company->currency);
        if ($currency === $companyCurrency) {
            return [
                'converted_purchase_amount' => null,
                'converted_to_currency' => null,
                'converted_at' => null,
                'exchange_rate' => null,
            ];
        }

        $date = $attrs['purchased_at'] ?? now()->format('Y-m-d');
        if (is_object($date) && method_exists($date, 'format')) {
            $date = $date->format('Y-m-d');
        }

        try {
            $rate = $this->frankfurter->rate((string) $date, $currency, $companyCurrency);

            return [
                'converted_purchase_amount' => (int) round((int) $amount * $rate),
                'converted_to_currency' => $companyCurrency,
                'converted_at' => now(),
                'exchange_rate' => $rate,
            ];
        } catch (Throwable) {
            return [
                'converted_purchase_amount' => null,
                'converted_to_currency' => null,
                'converted_at' => null,
                'exchange_rate' => null,
            ];
        }
    }

    private function assertEmployeeInCompany(Company $company, string $employeeId): void
    {
        $exists = Employee::query()
            ->where('company_id', $company->id)
            ->where('id', $employeeId)
            ->exists();
        if (! $exists) {
            throw new RuntimeException('Employee not found', 404);
        }
    }
}
