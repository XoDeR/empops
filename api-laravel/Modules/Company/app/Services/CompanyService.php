<?php

namespace Modules\Company\Services;

use App\Models\User;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Str;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use RuntimeException;

final class CompanyService
{
    /**
     * @return array{company: Company, employee: Employee}
     */
    public function create(User $user, string $name, string $currency = 'EUR'): array
    {
        return DB::transaction(function () use ($user, $name, $currency) {
            $company = Company::query()->create([
                'name' => $name,
                'slug' => $this->uniqueSlug($name),
                'currency' => strtoupper($currency),
                'code_to_join_company' => $this->uniqueJoinCode(),
            ]);

            $parts = $this->splitName($user->name);

            $employee = Employee::query()->create([
                'company_id' => $company->id,
                'user_id' => $user->id,
                'email' => $user->email,
                'first_name' => $parts['first_name'],
                'last_name' => $parts['last_name'],
                'hired_at' => now()->toDateString(),
                'locked' => false,
            ]);

            $employee->assignRole('administrator');

            return compact('company', 'employee');
        });
    }

    /**
     * @return array{company: Company, employee: Employee}
     */
    public function join(User $user, string $code): array
    {
        $company = Company::query()->where('code_to_join_company', $code)->first();
        if ($company === null) {
            throw new RuntimeException('Invalid join code', 404);
        }

        $existing = Employee::query()
            ->where('company_id', $company->id)
            ->where('user_id', $user->id)
            ->first();

        if ($existing !== null) {
            throw new RuntimeException('Already a member of this company', 409);
        }

        $emailTaken = Employee::query()
            ->where('company_id', $company->id)
            ->where('email', $user->email)
            ->whereNull('user_id')
            ->first();

        if ($emailTaken !== null) {
            throw new RuntimeException('An invitation exists for this email; accept the invite instead', 409);
        }

        $parts = $this->splitName($user->name);

        $employee = Employee::query()->create([
            'company_id' => $company->id,
            'user_id' => $user->id,
            'email' => $user->email,
            'first_name' => $parts['first_name'],
            'last_name' => $parts['last_name'],
            'hired_at' => now()->toDateString(),
            'locked' => false,
        ]);

        $employee->assignRole('employee');

        return compact('company', 'employee');
    }

    public function updateSettings(Company $company, array $data): Company
    {
        if (isset($data['name'])) {
            $company->name = $data['name'];
            $company->slug = $this->uniqueSlug($data['name'], (string) $company->id);
        }

        if (isset($data['currency'])) {
            $company->currency = strtoupper($data['currency']);
        }

        $company->save();

        return $company->fresh();
    }

    /**
     * @return array{id: string, name: string, slug: string, currency: string, code_to_join_company: string}
     */
    public function companyPayload(Company $company, bool $includeJoinCode = false): array
    {
        $company->loadMissing('media');

        $payload = [
            'id' => (string) $company->id,
            'name' => $company->name,
            'slug' => $company->slug,
            'currency' => $company->currency,
            'logo_url' => $company->getFirstMediaUrl('logo') ?: null,
        ];

        if ($includeJoinCode) {
            $payload['code_to_join_company'] = $company->code_to_join_company;
        }

        return $payload;
    }

    private function uniqueSlug(string $name, ?string $exceptId = null): string
    {
        $base = Str::slug($name) ?: 'company';
        $slug = $base;
        $i = 1;

        while (
            Company::query()
                ->when($exceptId, fn ($q) => $q->where('id', '!=', $exceptId))
                ->where('slug', $slug)
                ->exists()
        ) {
            $slug = $base.'-'.$i;
            $i++;
        }

        return $slug;
    }

    private function uniqueJoinCode(): string
    {
        do {
            $code = strtoupper(Str::random(8));
        } while (Company::query()->where('code_to_join_company', $code)->exists());

        return $code;
    }

    /**
     * @return array{first_name: string, last_name: string}
     */
    private function splitName(string $name): array
    {
        $parts = preg_split('/\s+/', trim($name), 2) ?: [];

        return [
            'first_name' => $parts[0] !== '' ? $parts[0] : 'User',
            'last_name' => $parts[1] ?? '',
        ];
    }
}
