<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('flows', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->string('name');
            $table->string('type');
            $table->timestampsTz();

            $table->index(['company_id', 'type']);
        });

        Schema::create('flow_steps', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('flow_id')->constrained('flows')->cascadeOnDelete();
            $table->unsignedInteger('number')->default(0);
            $table->string('unit_of_time')->default('days');
            $table->string('modifier');
            $table->integer('real_number_of_days')->default(0);
            $table->timestampsTz();

            $table->index(['flow_id', 'real_number_of_days']);
        });

        Schema::create('flow_actions', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('step_id')->constrained('flow_steps')->cascadeOnDelete();
            $table->string('type');
            $table->string('recipient');
            $table->text('specific_recipient_information')->nullable();
            $table->timestampsTz();
        });

        Schema::create('flow_action_runs', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('flow_action_id')->constrained('flow_actions')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->date('due_on');
            $table->timestampTz('executed_at')->nullable();
            $table->timestampsTz();

            $table->unique(['flow_action_id', 'employee_id', 'due_on'], 'flow_action_runs_unique');
            $table->index(['company_id', 'due_on', 'executed_at']);
        });

        Schema::create('wikis', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->string('title');
            $table->timestampsTz();

            $table->index('company_id');
        });

        Schema::create('wiki_pages', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('wiki_id')->constrained('wikis')->cascadeOnDelete();
            $table->string('title');
            $table->text('content')->nullable();
            $table->unsignedInteger('pageviews_counter')->default(0);
            $table->timestampsTz();

            $table->index('wiki_id');
        });

        Schema::create('wiki_page_revisions', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('page_id')->constrained('wiki_pages')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('employee_name');
            $table->string('title');
            $table->text('content')->nullable();
            $table->timestampsTz();

            $table->index('page_id');
        });

        Schema::create('ask_me_anything_sessions', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->date('happened_at');
            $table->boolean('active')->default(false);
            $table->string('theme')->nullable();
            $table->timestampsTz();

            $table->index(['company_id', 'active']);
        });

        Schema::create('ask_me_anything_questions', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('ask_me_anything_session_id')->constrained('ask_me_anything_sessions')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->text('question');
            $table->boolean('answered')->default(false);
            $table->boolean('anonymous')->default(false);
            $table->timestampsTz();

            $table->index('ask_me_anything_session_id');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('ask_me_anything_questions');
        Schema::dropIfExists('ask_me_anything_sessions');
        Schema::dropIfExists('wiki_page_revisions');
        Schema::dropIfExists('wiki_pages');
        Schema::dropIfExists('wikis');
        Schema::dropIfExists('flow_action_runs');
        Schema::dropIfExists('flow_actions');
        Schema::dropIfExists('flow_steps');
        Schema::dropIfExists('flows');
    }
};
