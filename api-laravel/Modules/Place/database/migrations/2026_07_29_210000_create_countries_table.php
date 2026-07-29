<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;
use Illuminate\Support\Str;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('countries', function (Blueprint $table) {
            $table->uuid('id')->primary();
            $table->string('name');
            $table->string('code', 2)->nullable();
            $table->timestamps();

            $table->unique('name');
            $table->unique('code');
        });

        $now = now();
        $rows = [
            ['United States', 'US'],
            ['United Kingdom', 'GB'],
            ['Canada', 'CA'],
            ['Germany', 'DE'],
            ['France', 'FR'],
            ['Netherlands', 'NL'],
            ['Belgium', 'BE'],
            ['Spain', 'ES'],
            ['Italy', 'IT'],
            ['Switzerland', 'CH'],
            ['Austria', 'AT'],
            ['Poland', 'PL'],
            ['Sweden', 'SE'],
            ['Norway', 'NO'],
            ['Denmark', 'DK'],
            ['Ireland', 'IE'],
            ['Portugal', 'PT'],
            ['Australia', 'AU'],
            ['New Zealand', 'NZ'],
            ['Romania', 'RO'],
        ];

        foreach ($rows as [$name, $code]) {
            DB::table('countries')->insert([
                'id' => (string) Str::uuid(),
                'name' => $name,
                'code' => $code,
                'created_at' => $now,
                'updated_at' => $now,
            ]);
        }
    }

    public function down(): void
    {
        Schema::dropIfExists('countries');
    }
};
