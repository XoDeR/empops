<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('positions', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->string('title');
            $table->timestamps();

            $table->unique(['company_id', 'title']);
        });

        Schema::create('employee_statuses', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->string('name');
            $table->string('type')->default('internal');
            $table->timestamps();

            $table->unique(['company_id', 'name']);
        });

        Schema::create('employees', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('user_id')->nullable()->constrained('users')->nullOnDelete();
            $table->string('email');
            $table->string('first_name');
            $table->string('last_name');
            $table->date('hired_at')->nullable();
            $table->foreignUuid('position_id')->nullable()->constrained('positions')->nullOnDelete();
            $table->foreignUuid('employee_status_id')->nullable()->constrained('employee_statuses')->nullOnDelete();
            $table->uuid('invitation_link')->nullable()->unique();
            $table->timestampTz('invitation_used_at')->nullable();
            $table->boolean('locked')->default(false);
            $table->timestamps();

            $table->unique(['company_id', 'email']);
            $table->unique(['company_id', 'user_id']);
            $table->index('user_id');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('employees');
        Schema::dropIfExists('employee_statuses');
        Schema::dropIfExists('positions');
    }
};
