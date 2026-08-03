<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('company_pto_policies', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->unsignedSmallInteger('year');
            $table->unsignedSmallInteger('total_worked_days')->default(0);
            $table->decimal('default_amount_of_allowed_holidays', 8, 2)->default(0);
            $table->decimal('default_amount_of_sick_days', 8, 2)->default(0);
            $table->decimal('default_amount_of_pto_days', 8, 2)->default(0);
            $table->timestampsTz();

            $table->unique(['company_id', 'year']);
        });

        Schema::create('company_calendars', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_pto_policy_id')->constrained('company_pto_policies')->cascadeOnDelete();
            $table->date('day');
            $table->unsignedTinyInteger('day_of_week');
            $table->unsignedSmallInteger('day_of_year');
            $table->boolean('is_worked')->default(true);
            $table->timestampsTz();

            $table->unique(['company_pto_policy_id', 'day']);
        });

        Schema::table('employees', function (Blueprint $table) {
            $table->decimal('holiday_balance', 10, 4)->default(0);
            $table->decimal('amount_of_allowed_holidays', 8, 2)->nullable();
            $table->decimal('amount_of_sick_days', 8, 2)->nullable();
            $table->decimal('amount_of_pto_days', 8, 2)->nullable();
        });

        Schema::create('employee_planned_holidays', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->date('planned_date');
            $table->string('type');
            $table->boolean('full')->default(true);
            $table->boolean('actually_taken')->default(false);
            $table->timestampsTz();

            $table->unique(['employee_id', 'planned_date', 'type']);
            $table->index('planned_date');
        });

        Schema::create('employee_daily_calendar_entries', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->date('log_date');
            $table->decimal('new_balance', 10, 4);
            $table->decimal('daily_accrued_amount', 10, 6);
            $table->decimal('current_holidays_per_year', 8, 2)->nullable();
            $table->decimal('default_amount_of_allowed_holidays_in_company', 8, 2)->nullable();
            $table->timestampsTz();

            $table->unique(['employee_id', 'log_date']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('employee_daily_calendar_entries');
        Schema::dropIfExists('employee_planned_holidays');

        Schema::table('employees', function (Blueprint $table) {
            $table->dropColumn([
                'holiday_balance',
                'amount_of_allowed_holidays',
                'amount_of_sick_days',
                'amount_of_pto_days',
            ]);
        });

        Schema::dropIfExists('company_calendars');
        Schema::dropIfExists('company_pto_policies');
    }
};
