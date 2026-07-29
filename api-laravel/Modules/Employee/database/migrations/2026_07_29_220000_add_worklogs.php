<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('employees', function (Blueprint $table) {
            $table->unsignedInteger('consecutive_worklog_missed')->default(0);
        });

        Schema::create('worklogs', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->text('content');
            $table->date('logged_on');
            $table->timestamps();

            $table->unique(['employee_id', 'logged_on']);
            $table->index('company_id');
            $table->index(['employee_id', 'logged_on']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('worklogs');
        Schema::table('employees', function (Blueprint $table) {
            $table->dropColumn('consecutive_worklog_missed');
        });
    }
};
