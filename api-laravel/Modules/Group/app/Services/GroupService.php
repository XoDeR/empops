<?php

namespace Modules\Group\Services;

use Illuminate\Support\Collection;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Group\Models\AgendaItem;
use Modules\Group\Models\Group;
use Modules\Group\Models\Meeting;
use Modules\Group\Models\MeetingDecision;

final class GroupService
{
    public function list(Company $company): Collection
    {
        return Group::query()->with(['members', 'meetings.agendaItems.decisions'])->where('company_id', $company->id)->orderBy('name')->get();
    }

    public function create(Company $company, array $data): Group
    {
        return Group::query()->create(['company_id' => $company->id, ...$data])->load('members');
    }

    public function update(Group $group, array $data): Group { $group->fill($data)->save(); return $group->fresh(['members', 'meetings']); }
    public function delete(Group $group): void { $group->delete(); }
    public function addMember(Group $group, Employee $employee): Group { $group->members()->syncWithoutDetaching([$employee->id]); return $group->fresh('members'); }
    public function removeMember(Group $group, Employee $employee): Group { $group->members()->detach($employee->id); return $group->fresh('members'); }

    public function createMeeting(Group $group, array $data): Meeting
    {
        return Meeting::query()->create(['group_id' => $group->id, ...$data])->load(['attendees', 'agendaItems']);
    }

    public function updateMeeting(Meeting $meeting, array $data): Meeting { $meeting->fill($data)->save(); return $meeting->fresh(['attendees', 'agendaItems.decisions']); }
    public function deleteMeeting(Meeting $meeting): void { $meeting->delete(); }

    public function markHappened(Meeting $meeting, ?string $date = null): Meeting
    {
        $meeting->update(['happened' => true, 'happened_at' => $date ?: now()->toDateString()]);

        return $meeting->fresh(['attendees', 'agendaItems.decisions']);
    }

    public function setAttendance(Meeting $meeting, Employee $employee, array $data): Meeting
    {
        $meeting->attendees()->syncWithoutDetaching([$employee->id => [
            'was_a_guest' => $data['was_a_guest'] ?? false,
            'attended' => $data['attended'] ?? false,
        ]]);

        return $meeting->fresh('attendees');
    }

    public function removeAttendance(Meeting $meeting, Employee $employee): Meeting
    {
        $meeting->attendees()->detach($employee->id);

        return $meeting->fresh('attendees');
    }

    public function createAgendaItem(Meeting $meeting, array $data): AgendaItem
    {
        $data['position'] ??= ((int) $meeting->agendaItems()->max('position')) + 1;

        return AgendaItem::query()->create(['meeting_id' => $meeting->id, ...$data]);
    }

    public function updateAgendaItem(AgendaItem $item, array $data): AgendaItem { $item->fill($data)->save(); return $item->fresh('decisions'); }
    public function deleteAgendaItem(AgendaItem $item): void { $item->delete(); }
    public function createDecision(AgendaItem $item, string $description): MeetingDecision { return MeetingDecision::query()->create(['agenda_item_id' => $item->id, 'description' => $description]); }
    public function updateDecision(MeetingDecision $decision, string $description): MeetingDecision { $decision->update(['description' => $description]); return $decision->fresh(); }
    public function deleteDecision(MeetingDecision $decision): void { $decision->delete(); }
}
