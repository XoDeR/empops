<?php

namespace Modules\Billing\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Modules\Company\Models\Company;

class CompanyInvoice extends Model
{
    use HasUuids;
    protected $fillable = ['company_id', 'usage_history_id', 'sent_to_customer', 'customer_has_paid', 'email_address_invoice_sent_to'];
    protected function casts(): array { return ['sent_to_customer' => 'boolean', 'customer_has_paid' => 'boolean']; }
    public function company(): BelongsTo { return $this->belongsTo(Company::class); }
    public function usage(): BelongsTo { return $this->belongsTo(CompanyDailyUsageHistory::class, 'usage_history_id'); }
}
