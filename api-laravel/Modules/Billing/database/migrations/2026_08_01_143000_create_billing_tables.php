<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('company_daily_usage_history', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->unsignedInteger('number_of_active_employees')->default(0);
            $table->date('logged_on');
            $table->timestampsTz();

            $table->unique(['company_id', 'logged_on']);
        });

        Schema::create('company_usage_history_details', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('usage_history_id')->constrained('company_daily_usage_history')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('employee_name')->nullable();
            $table->string('employee_email')->nullable();
            $table->timestampsTz();

            $table->index('usage_history_id');
        });

        Schema::create('company_invoices', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('usage_history_id')->nullable()->constrained('company_daily_usage_history')->nullOnDelete();
            $table->boolean('sent_to_customer')->default(false);
            $table->boolean('customer_has_paid')->default(false);
            $table->string('email_address_invoice_sent_to')->nullable();
            $table->timestampsTz();

            $table->index('company_id');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('company_invoices');
        Schema::dropIfExists('company_usage_history_details');
        Schema::dropIfExists('company_daily_usage_history');
    }
};
