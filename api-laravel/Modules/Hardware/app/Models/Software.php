<?php

namespace Modules\Hardware\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Spatie\MediaLibrary\HasMedia;
use Spatie\MediaLibrary\InteractsWithMedia;

class Software extends Model implements HasMedia
{
    use HasUuids;
    use InteractsWithMedia;

    protected $table = 'softwares';

    protected $fillable = [
        'company_id',
        'name',
        'product_key',
        'seats',
        'website',
        'licensed_to_name',
        'licensed_to_email_address',
        'order_number',
        'purchase_amount',
        'currency',
        'converted_purchase_amount',
        'converted_to_currency',
        'converted_at',
        'exchange_rate',
        'purchased_at',
    ];

    protected function casts(): array
    {
        return [
            'seats' => 'integer',
            'purchase_amount' => 'integer',
            'converted_purchase_amount' => 'integer',
            'exchange_rate' => 'decimal:8',
            'converted_at' => 'datetime',
            'purchased_at' => 'date',
        ];
    }

    public function registerMediaCollections(): void
    {
        $this->addMediaCollection('software');
    }

    public function getMorphClass(): string
    {
        return 'software';
    }

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function employees(): BelongsToMany
    {
        return $this->belongsToMany(Employee::class, 'employee_software')
            ->withPivot(['product_key', 'notes'])
            ->withTimestamps();
    }
}
