<?php

namespace Modules\Time\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;

class EmployeeWorkFromHome extends Model
{
    use HasUuids;

    protected $table = 'employee_work_from_home';

    protected $fillable = ['company_id', 'employee_id', 'date'];

    protected function casts(): array
    {
        return ['date' => 'date'];
    }
}
