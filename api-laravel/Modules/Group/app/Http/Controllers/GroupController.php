<?php

namespace Modules\Group\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Group\Models\AgendaItem;
use Modules\Group\Models\Group;
use Modules\Group\Models\Meeting;
use Modules\Group\Models\MeetingDecision;
use Modules\Group\Services\GroupService;

class GroupController extends Controller
{
    public function __construct(private readonly GroupService $groups) {}

    public function index(Request $r): JsonResponse { return ApiResponse::success($this->groups->list($this->company($r))); }
    public function show(Request $r, string $companyId, string $groupId): JsonResponse { return ApiResponse::success($this->group($r, $groupId)->load(['members', 'meetings.attendees', 'meetings.agendaItems.decisions'])); }

    public function store(Request $r): JsonResponse
    {
        $data = $r->validate(['name' => ['required', 'string', 'max:255'], 'mission' => ['nullable', 'string']]);

        return ApiResponse::success($this->groups->create($this->company($r), $data), 'Group created', 201);
    }

    public function update(Request $r, string $companyId, string $groupId): JsonResponse
    {
        $group = $this->group($r, $groupId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);
        $data = $r->validate(['name' => ['sometimes', 'required', 'string', 'max:255'], 'mission' => ['nullable', 'string']]);

        return ApiResponse::success($this->groups->update($group, $data), 'Group updated');
    }

    public function destroy(Request $r, string $companyId, string $groupId): JsonResponse
    {
        $group = $this->group($r, $groupId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);
        $this->groups->delete($group);

        return ApiResponse::success(null, 'Group deleted');
    }

    public function addMember(Request $r, string $companyId, string $groupId, string $employeeId): JsonResponse
    {
        $group = $this->group($r, $groupId);
        if (! $this->canManage($r, $group)) {
            return ApiResponse::error('Forbidden', 403);
        }

        return ApiResponse::success($this->groups->addMember($group, $this->employee($r, $employeeId)), 'Member added');
    }

    public function listMeetings(Request $r, string $companyId, string $groupId): JsonResponse
    {
        $group = $this->group($r, $groupId);

        return ApiResponse::success($group->meetings()->with(['attendees', 'agendaItems.decisions'])->orderByDesc('happened_at')->get());
    }

    public function showMeeting(Request $r, string $companyId, string $groupId, string $meetingId): JsonResponse
    {
        [, $meeting] = $this->meeting($r, $groupId, $meetingId);

        return ApiResponse::success($meeting->load(['attendees', 'agendaItems.decisions']));
    }

    public function removeMember(Request $r, string $companyId, string $groupId, string $employeeId): JsonResponse
    {
        $group = $this->group($r, $groupId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);

        return ApiResponse::success($this->groups->removeMember($group, $this->employee($r, $employeeId)), 'Member removed');
    }

    public function createMeeting(Request $r, string $companyId, string $groupId): JsonResponse
    {
        $group = $this->group($r, $groupId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);
        $data = $r->validate(['happened' => ['sometimes', 'boolean'], 'happened_at' => ['nullable', 'date']]);

        return ApiResponse::success($this->groups->createMeeting($group, $data), 'Meeting created', 201);
    }

    public function updateMeeting(Request $r, string $companyId, string $groupId, string $meetingId): JsonResponse
    {
        [$group, $meeting] = $this->meeting($r, $groupId, $meetingId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);
        $data = $r->validate(['happened' => ['sometimes', 'boolean'], 'happened_at' => ['nullable', 'date']]);

        return ApiResponse::success($this->groups->updateMeeting($meeting, $data), 'Meeting updated');
    }

    public function deleteMeeting(Request $r, string $companyId, string $groupId, string $meetingId): JsonResponse
    {
        [$group, $meeting] = $this->meeting($r, $groupId, $meetingId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);
        $this->groups->deleteMeeting($meeting);

        return ApiResponse::success(null, 'Meeting deleted');
    }

    public function happened(Request $r, string $companyId, string $groupId, string $meetingId): JsonResponse
    {
        [$group, $meeting] = $this->meeting($r, $groupId, $meetingId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);
        $data = $r->validate(['happened_at' => ['nullable', 'date']]);

        return ApiResponse::success($this->groups->markHappened($meeting, $data['happened_at'] ?? null), 'Meeting marked happened');
    }

    public function attendance(Request $r, string $companyId, string $groupId, string $meetingId): JsonResponse
    {
        [$group, $meeting] = $this->meeting($r, $groupId, $meetingId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);
        $data = $r->validate(['employee_id' => ['required', 'uuid'], 'was_a_guest' => ['sometimes', 'boolean'], 'attended' => ['sometimes', 'boolean']]);

        return ApiResponse::success($this->groups->setAttendance($meeting, $this->employee($r, $data['employee_id']), $data), 'Attendance saved');
    }

    public function removeAttendance(Request $r, string $companyId, string $groupId, string $meetingId, string $employeeId): JsonResponse
    {
        [$group, $meeting] = $this->meeting($r, $groupId, $meetingId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);

        return ApiResponse::success($this->groups->removeAttendance($meeting, $this->employee($r, $employeeId)), 'Attendance removed');
    }

    public function createAgenda(Request $r, string $companyId, string $groupId, string $meetingId): JsonResponse
    {
        [$group, $meeting] = $this->meeting($r, $groupId, $meetingId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);
        $data = $r->validate(['position' => ['sometimes', 'integer', 'min:0'], 'checked' => ['sometimes', 'boolean'], 'summary' => ['required', 'string', 'max:255'], 'description' => ['nullable', 'string'], 'presented_by_id' => ['nullable', 'uuid']]);
        if (isset($data['presented_by_id'])) $this->employee($r, $data['presented_by_id']);

        return ApiResponse::success($this->groups->createAgendaItem($meeting, $data), 'Agenda item created', 201);
    }

    public function updateAgenda(Request $r, string $companyId, string $groupId, string $meetingId, string $itemId): JsonResponse
    {
        [$group, $meeting] = $this->meeting($r, $groupId, $meetingId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);
        $item = AgendaItem::query()->where('meeting_id', $meeting->id)->findOrFail($itemId);
        $data = $r->validate(['position' => ['sometimes', 'integer', 'min:0'], 'checked' => ['sometimes', 'boolean'], 'summary' => ['sometimes', 'string', 'max:255'], 'description' => ['nullable', 'string'], 'presented_by_id' => ['nullable', 'uuid']]);

        return ApiResponse::success($this->groups->updateAgendaItem($item, $data), 'Agenda item updated');
    }

    public function deleteAgenda(Request $r, string $companyId, string $groupId, string $meetingId, string $itemId): JsonResponse
    {
        [$group, $meeting] = $this->meeting($r, $groupId, $meetingId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);
        $item = AgendaItem::query()->where('meeting_id', $meeting->id)->findOrFail($itemId);
        $this->groups->deleteAgendaItem($item);

        return ApiResponse::success(null, 'Agenda item deleted');
    }

    public function createDecision(Request $r, string $companyId, string $groupId, string $meetingId, string $itemId): JsonResponse
    {
        [$group, $meeting] = $this->meeting($r, $groupId, $meetingId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);
        $item = AgendaItem::query()->where('meeting_id', $meeting->id)->findOrFail($itemId);
        $data = $r->validate(['description' => ['required', 'string']]);

        return ApiResponse::success($this->groups->createDecision($item, $data['description']), 'Decision created', 201);
    }

    public function updateDecision(Request $r, string $companyId, string $groupId, string $meetingId, string $itemId, string $decisionId): JsonResponse
    {
        [$group, $meeting] = $this->meeting($r, $groupId, $meetingId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);
        $item = AgendaItem::query()->where('meeting_id', $meeting->id)->findOrFail($itemId);
        $decision = MeetingDecision::query()->where('agenda_item_id', $item->id)->findOrFail($decisionId);
        $data = $r->validate(['description' => ['required', 'string']]);

        return ApiResponse::success($this->groups->updateDecision($decision, $data['description']), 'Decision updated');
    }

    public function deleteDecision(Request $r, string $companyId, string $groupId, string $meetingId, string $itemId, string $decisionId): JsonResponse
    {
        [$group, $meeting] = $this->meeting($r, $groupId, $meetingId);
        if (! $this->canManage($r, $group)) return ApiResponse::error('Forbidden', 403);
        $item = AgendaItem::query()->where('meeting_id', $meeting->id)->findOrFail($itemId);
        $decision = MeetingDecision::query()->where('agenda_item_id', $item->id)->findOrFail($decisionId);
        $this->groups->deleteDecision($decision);

        return ApiResponse::success(null, 'Decision deleted');
    }

    private function company(Request $r): Company { return $r->attributes->get('company'); }
    private function actor(Request $r): Employee { return $r->attributes->get('employee'); }
    private function group(Request $r, string $id): Group { return Group::query()->where('company_id', $this->company($r)->id)->findOrFail($id); }
    private function employee(Request $r, string $id): Employee { return Employee::query()->where('company_id', $this->company($r)->id)->findOrFail($id); }
    private function canManage(Request $r, Group $group): bool { $actor = $this->actor($r); return $actor->hasPermissionTo('groups.manage') || $group->members()->where('employees.id', $actor->id)->exists(); }
    private function meeting(Request $r, string $groupId, string $meetingId): array { $group = $this->group($r, $groupId); return [$group, Meeting::query()->where('group_id', $group->id)->findOrFail($meetingId)]; }
}
