<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('places', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->uuid('placable_id');
            $table->string('placable_type');
            $table->string('street')->nullable();
            $table->string('city')->nullable();
            $table->string('province')->nullable();
            $table->string('postal_code')->nullable();
            $table->foreignUuid('country_id')->nullable()->constrained('countries')->nullOnDelete();
            $table->double('latitude')->nullable();
            $table->double('longitude')->nullable();
            $table->boolean('is_active')->default(false);
            $table->timestamps();

            $table->index(['placable_type', 'placable_id']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('places');
    }
};
