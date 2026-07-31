<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('companies', function (Blueprint $table) {
            $table->boolean('e_coffee_enabled')->default(false);
        });

        Schema::create('morales', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->unsignedTinyInteger('emotion');
            $table->string('comment')->nullable();
            $table->timestamps();

            $table->index(['employee_id', 'created_at']);
            $table->index(['company_id', 'created_at']);
        });

        Schema::create('morale_company_histories', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->double('average')->default(0);
            $table->unsignedInteger('number_of_employees')->default(0);
            $table->timestamps();

            $table->index(['company_id', 'created_at']);
        });

        Schema::create('morale_team_histories', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('team_id')->constrained('teams')->cascadeOnDelete();
            $table->double('average')->default(0);
            $table->unsignedInteger('number_of_team_members')->default(0);
            $table->timestamps();

            $table->index(['team_id', 'created_at']);
        });

        Schema::create('one_on_one_entries', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('manager_id')->constrained('employees')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->boolean('happened')->default(false);
            $table->timestampTz('happened_at')->nullable();
            $table->timestamps();

            $table->index(['manager_id', 'happened']);
            $table->index(['employee_id', 'happened']);
            $table->index(['company_id', 'happened']);
        });

        Schema::create('one_on_one_talking_points', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('one_on_one_entry_id')->constrained('one_on_one_entries')->cascadeOnDelete();
            $table->string('description');
            $table->boolean('checked')->default(false);
            $table->timestamps();
        });

        Schema::create('one_on_one_action_items', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('one_on_one_entry_id')->constrained('one_on_one_entries')->cascadeOnDelete();
            $table->string('description');
            $table->boolean('checked')->default(false);
            $table->timestamps();
        });

        Schema::create('one_on_one_notes', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('one_on_one_entry_id')->constrained('one_on_one_entries')->cascadeOnDelete();
            $table->text('note');
            $table->timestamps();
        });

        Schema::create('rate_your_manager_surveys', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('manager_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->boolean('active')->default(true);
            $table->timestampTz('valid_until_at')->nullable();
            $table->timestamps();

            $table->index(['company_id', 'active']);
            $table->index(['manager_id', 'active']);
        });

        Schema::create('rate_your_manager_answers', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('rate_your_manager_survey_id')
                ->constrained('rate_your_manager_surveys')
                ->cascadeOnDelete();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->boolean('active')->default(true);
            $table->string('rating')->nullable();
            $table->text('comment')->nullable();
            $table->boolean('reveal_identity_to_manager')->default(false);
            $table->timestamps();

            $table->unique(['rate_your_manager_survey_id', 'employee_id'], 'rym_answer_unique');
            $table->index(['employee_id', 'active']);
        });

        Schema::create('skills', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->string('name');
            $table->timestamps();

            $table->unique(['company_id', 'name']);
        });

        Schema::create('employee_skill', function (Blueprint $table) {
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->foreignUuid('skill_id')->constrained('skills')->cascadeOnDelete();
            $table->timestamps();
            $table->primary(['employee_id', 'skill_id']);
        });

        Schema::create('e_coffees', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->unsignedInteger('batch_number')->default(1);
            $table->boolean('active')->default(false);
            $table->timestamps();

            $table->index(['company_id', 'active']);
        });

        Schema::create('e_coffee_matches', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('e_coffee_id')->constrained('e_coffees')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->foreignUuid('with_employee_id')->constrained('employees')->cascadeOnDelete();
            $table->boolean('happened')->default(false);
            $table->timestamps();

            $table->index(['e_coffee_id', 'employee_id']);
        });

        Schema::create('discipline_cases', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->foreignUuid('opened_by_employee_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('opened_by_employee_name')->nullable();
            $table->boolean('active')->default(true);
            $table->timestamps();

            $table->index(['company_id', 'active']);
            $table->index(['employee_id', 'active']);
        });

        Schema::create('discipline_events', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('discipline_case_id')->constrained('discipline_cases')->cascadeOnDelete();
            $table->foreignUuid('author_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('author_name');
            $table->date('happened_at');
            $table->text('description');
            $table->timestamps();

            $table->index('discipline_case_id');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('discipline_events');
        Schema::dropIfExists('discipline_cases');
        Schema::dropIfExists('e_coffee_matches');
        Schema::dropIfExists('e_coffees');
        Schema::dropIfExists('employee_skill');
        Schema::dropIfExists('skills');
        Schema::dropIfExists('rate_your_manager_answers');
        Schema::dropIfExists('rate_your_manager_surveys');
        Schema::dropIfExists('one_on_one_notes');
        Schema::dropIfExists('one_on_one_action_items');
        Schema::dropIfExists('one_on_one_talking_points');
        Schema::dropIfExists('one_on_one_entries');
        Schema::dropIfExists('morale_team_histories');
        Schema::dropIfExists('morale_company_histories');
        Schema::dropIfExists('morales');

        Schema::table('companies', function (Blueprint $table) {
            $table->dropColumn('e_coffee_enabled');
        });
    }
};
