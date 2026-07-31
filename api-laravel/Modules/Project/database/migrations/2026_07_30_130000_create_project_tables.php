<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('projects', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('project_lead_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('status')->default('created');
            $table->boolean('completed')->default(false);
            $table->string('name');
            $table->string('code')->nullable();
            $table->string('short_code')->nullable();
            $table->string('emoji', 16)->nullable();
            $table->string('summary')->nullable();
            $table->text('description')->nullable();
            $table->timestampTz('started_at')->nullable();
            $table->timestampTz('planned_finished_at')->nullable();
            $table->timestampTz('actually_finished_at')->nullable();
            $table->timestamps();
            $table->index(['company_id', 'status']);
        });

        Schema::create('employee_project', function (Blueprint $table) {
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->foreignUuid('project_id')->constrained('projects')->cascadeOnDelete();
            $table->string('role')->nullable();
            $table->timestamps();
            $table->primary(['employee_id', 'project_id']);
        });

        Schema::create('project_team', function (Blueprint $table) {
            $table->foreignUuid('project_id')->constrained('projects')->cascadeOnDelete();
            $table->foreignUuid('team_id')->constrained('teams')->cascadeOnDelete();
            $table->timestamps();
            $table->primary(['project_id', 'team_id']);
        });

        Schema::create('project_links', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('project_id')->constrained('projects')->cascadeOnDelete();
            $table->string('type');
            $table->string('label')->nullable();
            $table->string('url');
            $table->timestamps();
        });

        Schema::create('project_statuses', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('project_id')->constrained('projects')->cascadeOnDelete();
            $table->foreignUuid('author_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('title');
            $table->string('status');
            $table->text('description');
            $table->timestamps();
        });

        Schema::create('project_messages', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('project_id')->constrained('projects')->cascadeOnDelete();
            $table->foreignUuid('author_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('title');
            $table->text('content');
            $table->timestamps();
        });

        Schema::create('project_decisions', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('project_id')->constrained('projects')->cascadeOnDelete();
            $table->foreignUuid('author_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('title');
            $table->date('decided_at')->nullable();
            $table->timestamps();
        });

        Schema::create('project_decision_deciders', function (Blueprint $table) {
            $table->foreignUuid('project_decision_id')->constrained('project_decisions')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->timestamps();
            $table->primary(['project_decision_id', 'employee_id'], 'project_decision_deciders_pk');
        });

        Schema::create('project_task_lists', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('project_id')->constrained('projects')->cascadeOnDelete();
            $table->foreignUuid('author_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('title');
            $table->text('description')->nullable();
            $table->timestamps();
        });

        Schema::create('project_tasks', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('project_id')->constrained('projects')->cascadeOnDelete();
            $table->foreignUuid('project_task_list_id')->nullable()->constrained('project_task_lists')->cascadeOnDelete();
            $table->foreignUuid('author_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->foreignUuid('assignee_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('title');
            $table->text('description')->nullable();
            $table->boolean('completed')->default(false);
            $table->timestampTz('completed_at')->nullable();
            $table->timestamps();
        });

        Schema::create('comments', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('author_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('author_name');
            $table->text('content');
            $table->uuid('commentable_id')->nullable();
            $table->string('commentable_type')->nullable();
            $table->timestamps();
            $table->index(['commentable_type', 'commentable_id']);
        });

        Schema::create('issue_types', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->string('name');
            $table->string('icon')->nullable();
            $table->timestamps();
            $table->unique(['company_id', 'name']);
        });

        Schema::create('project_boards', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('project_id')->constrained('projects')->cascadeOnDelete();
            $table->string('name');
            $table->timestamps();
        });

        Schema::create('project_sprints', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('project_id')->constrained('projects')->cascadeOnDelete();
            $table->foreignUuid('project_board_id')->nullable()->constrained('project_boards')->nullOnDelete();
            $table->string('name');
            $table->boolean('active')->default(false);
            $table->unsignedInteger('position')->nullable();
            $table->timestampTz('started_at')->nullable();
            $table->timestampTz('completed_at')->nullable();
            $table->timestamps();
        });

        Schema::create('project_issues', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('project_id')->constrained('projects')->cascadeOnDelete();
            $table->foreignUuid('project_board_id')->nullable()->constrained('project_boards')->nullOnDelete();
            $table->foreignUuid('reporter_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->foreignUuid('issue_type_id')->nullable()->constrained('issue_types')->nullOnDelete();
            $table->boolean('is_separator')->default(false);
            $table->unsignedInteger('id_in_project');
            $table->string('key');
            $table->string('slug');
            $table->string('title');
            $table->mediumText('description')->nullable();
            $table->unsignedInteger('story_points')->nullable();
            $table->timestamps();
            $table->unique(['project_id', 'id_in_project']);
            $table->unique(['project_id', 'key']);
        });

        Schema::create('project_issue_project_sprint', function (Blueprint $table) {
            $table->foreignUuid('project_issue_id')->constrained('project_issues')->cascadeOnDelete();
            $table->foreignUuid('project_sprint_id')->constrained('project_sprints')->cascadeOnDelete();
            $table->unsignedInteger('position')->default(0);
            $table->primary(['project_issue_id', 'project_sprint_id'], 'project_issue_sprint_pk');
        });

        Schema::create('project_labels', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('project_id')->constrained('projects')->cascadeOnDelete();
            $table->string('name');
            $table->timestamps();
        });

        Schema::create('project_issue_project_label', function (Blueprint $table) {
            $table->foreignUuid('project_issue_id')->constrained('project_issues')->cascadeOnDelete();
            $table->foreignUuid('project_label_id')->constrained('project_labels')->cascadeOnDelete();
            $table->primary(['project_issue_id', 'project_label_id'], 'project_issue_label_pk');
        });

        Schema::create('project_issue_assignees', function (Blueprint $table) {
            $table->foreignUuid('project_issue_id')->constrained('project_issues')->cascadeOnDelete();
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->primary(['project_issue_id', 'employee_id'], 'project_issue_assignees_pk');
        });

        Schema::table('time_tracking_entries', function (Blueprint $table) {
            $table->dropUnique('time_entry_day_unique');
            $table->foreignUuid('project_id')->nullable()->after('employee_id')->constrained('projects')->nullOnDelete();
            $table->foreignUuid('project_task_id')->nullable()->after('project_id')->constrained('project_tasks')->nullOnDelete();
        });
    }

    public function down(): void
    {
        Schema::table('time_tracking_entries', function (Blueprint $table) {
            $table->dropConstrainedForeignId('project_task_id');
            $table->dropConstrainedForeignId('project_id');
            $table->unique(['timesheet_id', 'employee_id', 'happened_at'], 'time_entry_day_unique');
        });

        Schema::dropIfExists('project_issue_assignees');
        Schema::dropIfExists('project_issue_project_label');
        Schema::dropIfExists('project_labels');
        Schema::dropIfExists('project_issue_project_sprint');
        Schema::dropIfExists('project_issues');
        Schema::dropIfExists('project_sprints');
        Schema::dropIfExists('project_boards');
        Schema::dropIfExists('issue_types');
        Schema::dropIfExists('comments');
        Schema::dropIfExists('project_tasks');
        Schema::dropIfExists('project_task_lists');
        Schema::dropIfExists('project_decision_deciders');
        Schema::dropIfExists('project_decisions');
        Schema::dropIfExists('project_messages');
        Schema::dropIfExists('project_statuses');
        Schema::dropIfExists('project_links');
        Schema::dropIfExists('project_team');
        Schema::dropIfExists('employee_project');
        Schema::dropIfExists('projects');
    }
};
