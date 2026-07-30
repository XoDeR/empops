<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('expense_categories', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->string('name');
            $table->timestamps();

            $table->unique(['company_id', 'name']);
        });

        Schema::create('expenses', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->foreignUuid('expense_category_id')->nullable()->constrained('expense_categories')->nullOnDelete();
            $table->string('status');
            $table->string('title');
            $table->unsignedBigInteger('amount');
            $table->string('currency', 3);
            $table->unsignedBigInteger('converted_amount')->nullable();
            $table->string('converted_to_currency', 3)->nullable();
            $table->timestampTz('converted_at')->nullable();
            $table->decimal('exchange_rate', 18, 8)->nullable();
            $table->text('description')->nullable();
            $table->date('expensed_at');
            $table->foreignUuid('manager_approver_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->timestampTz('manager_approver_approved_at')->nullable();
            $table->text('manager_rejection_explanation')->nullable();
            $table->foreignUuid('accounting_approver_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->timestampTz('accounting_approver_approved_at')->nullable();
            $table->text('accounting_rejection_explanation')->nullable();
            $table->timestamps();

            $table->index(['company_id', 'status', 'created_at']);
            $table->index(['employee_id', 'status']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('expenses');
        Schema::dropIfExists('expense_categories');
    }
};
