<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('recruiting_stage_templates', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->string('name');
            $table->timestamps();
            $table->index('company_id');
        });

        Schema::create('recruiting_stages', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('recruiting_stage_template_id')
                ->constrained('recruiting_stage_templates')
                ->cascadeOnDelete();
            $table->string('name');
            $table->unsignedInteger('position')->default(0);
            $table->timestamps();
            $table->index('recruiting_stage_template_id');
        });

        Schema::create('job_openings', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('position_id')->constrained('positions')->cascadeOnDelete();
            $table->foreignUuid('recruiting_stage_template_id')
                ->nullable()
                ->constrained('recruiting_stage_templates')
                ->nullOnDelete();
            $table->foreignUuid('team_id')->nullable()->constrained('teams')->nullOnDelete();
            $table->uuid('fulfilled_by_candidate_id')->nullable();
            $table->string('title');
            $table->text('description');
            $table->string('slug');
            $table->string('reference_number')->nullable();
            $table->boolean('active')->default(false);
            $table->boolean('fulfilled')->default(false);
            $table->unsignedInteger('page_views')->default(0);
            $table->timestampTz('activated_at')->nullable();
            $table->timestampTz('fulfilled_at')->nullable();
            $table->timestamps();
            $table->unique(['company_id', 'slug']);
            $table->index(['company_id', 'active', 'fulfilled']);
        });

        Schema::create('job_opening_sponsor', function (Blueprint $table) {
            $table->foreignUuid('job_opening_id')->constrained('job_openings')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->timestamps();
            $table->primary(['job_opening_id', 'employee_id']);
        });

        Schema::create('candidates', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('job_opening_id')->constrained('job_openings')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('name');
            $table->string('email');
            $table->uuid('uuid')->unique();
            $table->string('url')->nullable();
            $table->string('desired_salary')->nullable();
            $table->text('notes')->nullable();
            $table->boolean('application_completed')->default(false);
            $table->boolean('rejected')->default(false);
            $table->string('employee_name')->nullable();
            $table->timestamps();
            $table->index(['job_opening_id', 'application_completed', 'rejected']);
        });

        Schema::table('job_openings', function (Blueprint $table) {
            $table->foreign('fulfilled_by_candidate_id')
                ->references('id')
                ->on('candidates')
                ->nullOnDelete();
        });

        Schema::create('candidate_stages', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('candidate_id')->constrained('candidates')->cascadeOnDelete();
            $table->foreignUuid('decider_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('stage_name');
            $table->unsignedInteger('stage_position')->default(0);
            $table->string('status')->default('pending');
            $table->string('decider_name')->nullable();
            $table->timestampTz('decided_at')->nullable();
            $table->timestamps();
            $table->index(['candidate_id', 'stage_position']);
        });

        Schema::create('candidate_stage_notes', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('candidate_stage_id')->constrained('candidate_stages')->cascadeOnDelete();
            $table->foreignUuid('author_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('author_name');
            $table->text('note');
            $table->timestamps();
        });

        Schema::create('candidate_stage_participants', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('candidate_stage_id')->constrained('candidate_stages')->cascadeOnDelete();
            $table->foreignUuid('participant_id')->constrained('employees')->cascadeOnDelete();
            $table->string('participant_name');
            $table->boolean('participated')->default(false);
            $table->timestampTz('participated_at')->nullable();
            $table->timestamps();
            $table->unique(['candidate_stage_id', 'participant_id'], 'candidate_stage_participant_unique');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('candidate_stage_participants');
        Schema::dropIfExists('candidate_stage_notes');
        Schema::dropIfExists('candidate_stages');
        Schema::table('job_openings', function (Blueprint $table) {
            $table->dropForeign(['fulfilled_by_candidate_id']);
        });
        Schema::dropIfExists('candidates');
        Schema::dropIfExists('job_opening_sponsor');
        Schema::dropIfExists('job_openings');
        Schema::dropIfExists('recruiting_stages');
        Schema::dropIfExists('recruiting_stage_templates');
    }
};
