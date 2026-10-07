<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (Schema::hasColumn('activity_log', 'attribute_changes')) {
            return;
        }

        Schema::table('activity_log', function (Blueprint $table) {
            $table->json('attribute_changes')->nullable()->after('causer_id');
        });
    }

    public function down(): void
    {
        if (! Schema::hasColumn('activity_log', 'attribute_changes')) {
            return;
        }

        Schema::table('activity_log', function (Blueprint $table) {
            $table->dropColumn('attribute_changes');
        });
    }
};
