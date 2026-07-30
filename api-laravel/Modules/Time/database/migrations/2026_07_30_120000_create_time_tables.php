<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('companies', function (Blueprint $table) {
            $table->boolean('work_from_home_enabled')->default(true);
        });

        Schema::create('timesheets', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->date('started_at');
            $table->date('ended_at');
            $table->string('status')->default('open');
            $table->foreignUuid('approver_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->timestampTz('approved_at')->nullable();
            $table->timestamps();

            $table->unique(['employee_id', 'started_at']);
            $table->index(['company_id', 'status', 'started_at']);
        });

        Schema::create('time_tracking_entries', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('timesheet_id')->constrained('timesheets')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->unsignedSmallInteger('duration');
            $table->date('happened_at');
            $table->text('description')->nullable();
            $table->timestamps();

            $table->unique(['timesheet_id', 'employee_id', 'happened_at'], 'time_entry_day_unique');
        });

        Schema::create('employee_work_from_home', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->date('date');
            $table->timestamps();

            $table->unique(['employee_id', 'date']);
            $table->index(['company_id', 'date']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('employee_work_from_home');
        Schema::dropIfExists('time_tracking_entries');
        Schema::dropIfExists('timesheets');

        Schema::table('companies', function (Blueprint $table) {
            $table->dropColumn('work_from_home_enabled');
        });
    }
};
