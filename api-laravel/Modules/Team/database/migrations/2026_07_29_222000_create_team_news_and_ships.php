<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('team_news', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('team_id')->constrained('teams')->cascadeOnDelete();
            $table->foreignUuid('author_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('author_name');
            $table->string('title');
            $table->text('content');
            $table->timestamps();

            $table->index(['team_id', 'created_at']);
        });

        Schema::create('ships', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->foreignUuid('company_id')->constrained('companies')->cascadeOnDelete();
            $table->foreignUuid('team_id')->constrained('teams')->cascadeOnDelete();
            $table->foreignUuid('author_id')->nullable()->constrained('employees')->nullOnDelete();
            $table->string('author_name');
            $table->string('title');
            $table->text('description')->nullable();
            $table->timestamps();

            $table->index(['team_id', 'created_at']);
        });

        Schema::create('employee_ship', function (Blueprint $table) {
            $table->foreignUuid('employee_id')->constrained('employees')->cascadeOnDelete();
            $table->foreignUuid('ship_id')->constrained('ships')->cascadeOnDelete();
            $table->timestamp('created_at')->useCurrent();

            $table->primary(['employee_id', 'ship_id']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('employee_ship');
        Schema::dropIfExists('ships');
        Schema::dropIfExists('team_news');
    }
};
