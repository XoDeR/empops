<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('hardware', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('name');
            $table->string('serial_number')->nullable();
            $table->timestampsTz();

            $table->index(['company_id', 'employee_id']);
        });

        Schema::create('softwares', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->string('name');
            $table->text('product_key')->nullable();
            $table->unsignedInteger('seats');
            $table->string('website')->nullable();
            $table->string('licensed_to_name')->nullable();
            $table->string('licensed_to_email_address')->nullable();
            $table->string('order_number')->nullable();
            $table->unsignedBigInteger('purchase_amount')->nullable();
            $table->string('currency', 3)->nullable();
            $table->unsignedBigInteger('converted_purchase_amount')->nullable();
            $table->string('converted_to_currency', 3)->nullable();
            $table->timestampTz('converted_at')->nullable();
            $table->decimal('exchange_rate', 18, 8)->nullable();
            $table->date('purchased_at')->nullable();
            $table->timestampsTz();

            $table->index('company_id');
        });

        Schema::create('employee_software', function (Blueprint $table) {
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->foreignUuid('software_id')->constrained('softwares')->cascadeOnDelete();
            $table->text('product_key')->nullable();
            $table->text('notes')->nullable();
            $table->timestampsTz();

            $table->primary(['employee_id', 'software_id']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('employee_software');
        Schema::dropIfExists('softwares');
        Schema::dropIfExists('hardware');
    }
};
