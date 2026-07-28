<?php

namespace Database\Seeders;

use App\Models\Permission;
use App\Models\Role;
use Illuminate\Database\Seeder;

class RolePermissionSeeder extends Seeder
{
    public function run(): void
    {
        $permissions = [
            'company.update',
            'adminland.access',
            'employees.view',
            'employees.create',
            'employees.update',
            'employees.delete',
            'employees.invite',
            'positions.view',
            'positions.create',
            'positions.update',
            'positions.delete',
            'employee-statuses.view',
            'employee-statuses.create',
            'employee-statuses.update',
            'employee-statuses.delete',
            'teams.view',
            'teams.create',
            'teams.update',
            'teams.delete',
            'teams.manage_members',
            'hierarchy.assign',
        ];

        foreach ($permissions as $name) {
            Permission::findOrCreate($name, 'web');
        }

        $admin = Role::findOrCreate('administrator', 'web');
        $hr = Role::findOrCreate('hr', 'web');
        $employee = Role::findOrCreate('employee', 'web');
        Role::findOrCreate('manager', 'web');

        $admin->syncPermissions($permissions);

        $hr->syncPermissions([
            'adminland.access',
            'employees.view',
            'employees.create',
            'employees.update',
            'employees.delete',
            'employees.invite',
            'positions.view',
            'positions.create',
            'positions.update',
            'positions.delete',
            'employee-statuses.view',
            'employee-statuses.create',
            'employee-statuses.update',
            'employee-statuses.delete',
            'teams.view',
            'teams.create',
            'teams.update',
            'teams.delete',
            'teams.manage_members',
            'hierarchy.assign',
        ]);

        $employee->syncPermissions([
            'employees.view',
            'positions.view',
            'employee-statuses.view',
            'teams.view',
        ]);
    }
}
