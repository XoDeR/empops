<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('groups', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->string('name');
            $table->text('mission')->nullable();
            $table->timestampsTz();

            $table->index('company_id');
        });

        Schema::create('employee_group', function (Blueprint $table) {
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->foreignUuid('group_id')->constrained('groups')->cascadeOnDelete();
            $table->timestampsTz();

            $table->primary(['employee_id', 'group_id']);
        });

        Schema::create('meetings', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('group_id')->constrained('groups')->cascadeOnDelete();
            $table->boolean('happened')->default(false);
            $table->date('happened_at')->nullable();
            $table->timestampsTz();

            $table->index('group_id');
        });

        Schema::create('employee_meeting', function (Blueprint $table) {
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->foreignUuid('meeting_id')->constrained('meetings')->cascadeOnDelete();
            $table->boolean('was_a_guest')->default(false);
            $table->boolean('attended')->default(false);
            $table->timestampsTz();

            $table->primary(['employee_id', 'meeting_id']);
        });

        Schema::create('agenda_items', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('meeting_id')->constrained('meetings')->cascadeOnDelete();
            $table->unsignedInteger('position')->default(0);
            $table->boolean('checked')->default(false);
            $table->string('summary');
            $table->text('description')->nullable();
            $table->foreignUuid('presented_by_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->timestampsTz();

            $table->index(['meeting_id', 'position']);
        });

        Schema::create('meeting_decisions', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('agenda_item_id')->constrained('agenda_items')->cascadeOnDelete();
            $table->text('description');
            $table->timestampsTz();

            $table->index('agenda_item_id');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('meeting_decisions');
        Schema::dropIfExists('agenda_items');
        Schema::dropIfExists('employee_meeting');
        Schema::dropIfExists('meetings');
        Schema::dropIfExists('employee_group');
        Schema::dropIfExists('groups');
    }
};
