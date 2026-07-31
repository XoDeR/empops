<?php

namespace Modules\Project\Services;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Str;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Project\Models\Comment;
use Modules\Project\Models\IssueType;
use Modules\Project\Models\Project;
use Modules\Project\Models\ProjectBoard;
use Modules\Project\Models\ProjectDecision;
use Modules\Project\Models\ProjectIssue;
use Modules\Project\Models\ProjectLink;
use Modules\Project\Models\ProjectMessage;
use Modules\Project\Models\ProjectSprint;
use Modules\Project\Models\ProjectStatus;
use Modules\Project\Models\ProjectTask;
use Modules\Project\Models\ProjectTaskList;
use Modules\Team\Models\Team;
use Modules\Time\Models\TimeTrackingEntry;
use Modules\Uploads\Services\MediaAttachService;
use RuntimeException;
use Spatie\MediaLibrary\MediaCollections\Models\Media;

final class ProjectService
{
    private const DEFAULT_ISSUE_TYPES = [
        ['name' => 'Bug', 'icon' => 'bug'],
        ['name' => 'Story', 'icon' => 'story'],
        ['name' => 'Task', 'icon' => 'task'],
        ['name' => 'Epic', 'icon' => 'epic'],
    ];

    public function __construct(
        private readonly MediaAttachService $mediaAttach,
    ) {}

    public function canAccess(Employee $employee, Project $project): bool
    {
        if ($employee->hasAnyRole(['administrator', 'hr'])) {
            return true;
        }

        if ($employee->can('projects.view')) {
            return true;
        }

        return $this->isMember($employee, $project);
    }

    public function canManage(Employee $employee, Project $project): bool
    {
        if ($employee->hasAnyRole(['administrator', 'hr'])) {
            return true;
        }

        if ($employee->can('projects.update') || $employee->can('projects.manage_members')) {
            return true;
        }

        return $this->isMember($employee, $project);
    }

    public function ensureIssueTypes(Company $company): void
    {
        if (IssueType::query()->where('company_id', $company->id)->exists()) {
            return;
        }

        foreach (self::DEFAULT_ISSUE_TYPES as $type) {
            IssueType::query()->create([
                'company_id' => $company->id,
                'name' => $type['name'],
                'icon' => $type['icon'],
            ]);
        }
    }

    public function listIssueTypes(Company $company): Collection
    {
        $this->ensureIssueTypes($company);

        return IssueType::query()
            ->where('company_id', $company->id)
            ->orderBy('name')
            ->get();
    }

    public function list(Company $company, Employee $actor): Collection
    {
        $query = Project::query()
            ->with(['lead', 'employees', 'teams'])
            ->where('company_id', $company->id);

        if (! $actor->hasAnyRole(['administrator', 'hr']) && ! $actor->can('projects.view')) {
            $query->whereHas('employees', fn (Builder $q) => $q->where('employees.id', $actor->id));
        }

        return $query->orderBy('name')->get();
    }

    public function find(Company $company, string $id): Project
    {
        return Project::query()
            ->with(['lead', 'employees', 'teams'])
            ->where('company_id', $company->id)
            ->where('id', $id)
            ->firstOrFail();
    }

    public function create(Company $company, Employee $actor, array $data): Project
    {
        return DB::transaction(function () use ($company, $actor, $data) {
            $shortCode = $data['short_code'] ?? null;
            if ($shortCode === null || $shortCode === '') {
                $shortCode = $this->generateShortCode($company, $data['name']);
            }

            $project = Project::query()->create([
                'company_id' => $company->id,
                'project_lead_id' => $data['project_lead_id'] ?? null,
                'name' => $data['name'],
                'code' => $data['code'] ?? null,
                'short_code' => $shortCode,
                'emoji' => $data['emoji'] ?? null,
                'summary' => $data['summary'] ?? null,
                'description' => $data['description'] ?? null,
                'status' => $data['status'] ?? Project::STATUS_CREATED,
                'started_at' => $data['started_at'] ?? null,
                'planned_finished_at' => $data['planned_finished_at'] ?? null,
            ]);

            $project->employees()->syncWithoutDetaching([
                (string) $actor->id => ['role' => null],
            ]);

            if (! empty($data['project_lead_id'])) {
                $project->employees()->syncWithoutDetaching([
                    (string) $data['project_lead_id'] => ['role' => 'lead'],
                ]);
            }

            return $project->fresh(['lead', 'employees', 'teams']);
        });
    }

    public function update(Project $project, Employee $actor, array $data): Project
    {
        $this->ensureManage($actor, $project);

        $status = $data['status'] ?? $project->status;

        $project->update([
            'name' => $data['name'] ?? $project->name,
            'code' => array_key_exists('code', $data) ? $data['code'] : $project->code,
            'short_code' => array_key_exists('short_code', $data) ? $data['short_code'] : $project->short_code,
            'emoji' => array_key_exists('emoji', $data) ? $data['emoji'] : $project->emoji,
            'summary' => array_key_exists('summary', $data) ? $data['summary'] : $project->summary,
            'description' => array_key_exists('description', $data) ? $data['description'] : $project->description,
            'status' => $status,
            'project_lead_id' => array_key_exists('project_lead_id', $data) ? $data['project_lead_id'] : $project->project_lead_id,
            'started_at' => array_key_exists('started_at', $data) ? $data['started_at'] : $project->started_at,
            'planned_finished_at' => array_key_exists('planned_finished_at', $data) ? $data['planned_finished_at'] : $project->planned_finished_at,
            'completed' => in_array($status, [Project::STATUS_CLOSED, Project::STATUS_CANCELLED], true),
        ]);

        return $project->fresh(['lead', 'employees', 'teams']);
    }

    public function delete(Project $project, Employee $actor): void
    {
        $this->ensureManage($actor, $project);
        $project->delete();
    }

    public function addMember(Project $project, Employee $actor, Employee $member, ?string $role = null): Project
    {
        $this->ensureManage($actor, $project);

        if ((string) $member->company_id !== (string) $project->company_id) {
            throw new RuntimeException('Employee does not belong to this company', 422);
        }

        $project->employees()->syncWithoutDetaching([
            (string) $member->id => ['role' => $role],
        ]);

        return $project->fresh(['lead', 'employees', 'teams']);
    }

    public function removeMember(Project $project, Employee $actor, Employee $member): Project
    {
        $this->ensureManage($actor, $project);
        $project->employees()->detach($member->id);

        if ((string) $project->project_lead_id === (string) $member->id) {
            $project->update(['project_lead_id' => null]);
        }

        return $project->fresh(['lead', 'employees', 'teams']);
    }

    public function setLead(Project $project, Employee $actor, ?Employee $lead): Project
    {
        $this->ensureManage($actor, $project);

        if ($lead !== null) {
            if ((string) $lead->company_id !== (string) $project->company_id) {
                throw new RuntimeException('Employee does not belong to this company', 422);
            }

            $project->employees()->syncWithoutDetaching([
                (string) $lead->id => ['role' => 'lead'],
            ]);
        }

        $project->update(['project_lead_id' => $lead?->id]);

        return $project->fresh(['lead', 'employees', 'teams']);
    }

    public function attachTeam(Project $project, Employee $actor, Team $team): Project
    {
        $this->ensureManage($actor, $project);

        if ((string) $team->company_id !== (string) $project->company_id) {
            throw new RuntimeException('Team does not belong to this company', 422);
        }

        $project->teams()->syncWithoutDetaching([(string) $team->id]);

        return $project->fresh(['lead', 'employees', 'teams']);
    }

    public function detachTeam(Project $project, Employee $actor, Team $team): Project
    {
        $this->ensureManage($actor, $project);
        $project->teams()->detach($team->id);

        return $project->fresh(['lead', 'employees', 'teams']);
    }

    public function listLinks(Project $project, Employee $actor): Collection
    {
        $this->ensureAccess($actor, $project);

        return $project->links()->orderBy('created_at')->get();
    }

    public function createLink(Project $project, Employee $actor, array $data): ProjectLink
    {
        $this->ensureManage($actor, $project);

        return $project->links()->create([
            'type' => $data['type'],
            'label' => $data['label'] ?? null,
            'url' => $data['url'],
        ]);
    }

    public function updateLink(ProjectLink $link, Employee $actor, array $data): ProjectLink
    {
        $this->ensureManage($actor, $link->project);
        $link->update([
            'type' => $data['type'] ?? $link->type,
            'label' => array_key_exists('label', $data) ? $data['label'] : $link->label,
            'url' => $data['url'] ?? $link->url,
        ]);

        return $link->fresh();
    }

    public function deleteLink(ProjectLink $link, Employee $actor): void
    {
        $this->ensureManage($actor, $link->project);
        $link->delete();
    }

    public function listStatuses(Project $project, Employee $actor): Collection
    {
        $this->ensureAccess($actor, $project);

        return $project->statuses()->with('author')->orderByDesc('created_at')->get();
    }

    public function createStatus(Project $project, Employee $actor, array $data): ProjectStatus
    {
        $this->ensureManage($actor, $project);

        return $project->statuses()->create([
            'author_id' => $actor->id,
            'title' => $data['title'],
            'status' => $data['status'],
            'description' => $data['description'],
        ])->load('author');
    }

    public function listFiles(Project $project, Employee $actor): Collection
    {
        $this->ensureAccess($actor, $project);

        return $project->getMedia('files');
    }

    public function attachFile(Project $project, Employee $actor, int $temporaryUploadId, int $mediaId): Media
    {
        $this->ensureManage($actor, $project);

        return $this->mediaAttach->attachFromTemporary(
            $project,
            'files',
            $temporaryUploadId,
            $mediaId,
            clearExisting: false,
        );
    }

    public function deleteFile(Project $project, Employee $actor, int $mediaId): void
    {
        $this->ensureManage($actor, $project);

        $media = $project->getMedia('files')->firstWhere('id', $mediaId);
        if ($media === null) {
            throw new RuntimeException('File not found', 404);
        }

        $media->delete();
    }

    public function listMessages(Project $project, Employee $actor): Collection
    {
        $this->ensureAccess($actor, $project);

        return $project->messages()->with(['author', 'comments'])->orderByDesc('created_at')->get();
    }

    public function findMessage(Project $project, string $messageId): ProjectMessage
    {
        return $project->messages()->with(['author', 'comments'])->where('id', $messageId)->firstOrFail();
    }

    public function createMessage(Project $project, Employee $actor, array $data): ProjectMessage
    {
        $this->ensureManage($actor, $project);

        return $project->messages()->create([
            'author_id' => $actor->id,
            'title' => $data['title'],
            'content' => $data['content'],
        ])->load(['author', 'comments']);
    }

    public function updateMessage(ProjectMessage $message, Employee $actor, array $data): ProjectMessage
    {
        $this->ensureManage($actor, $message->project);
        $message->update([
            'title' => $data['title'] ?? $message->title,
            'content' => $data['content'] ?? $message->content,
        ]);

        return $message->fresh(['author', 'comments']);
    }

    public function deleteMessage(ProjectMessage $message, Employee $actor): void
    {
        $this->ensureManage($actor, $message->project);
        $message->delete();
    }

    public function createMessageComment(
        Company $company,
        ProjectMessage $message,
        Employee $actor,
        array $data,
    ): Comment {
        $this->ensureManage($actor, $message->project);

        return $message->comments()->create([
            'company_id' => $company->id,
            'author_id' => $actor->id,
            'author_name' => $actor->fullName(),
            'content' => $data['content'],
        ]);
    }

    public function updateComment(Comment $comment, Employee $actor, array $data): Comment
    {
        $this->ensureCommentManage($comment, $actor);
        $comment->update(['content' => $data['content']]);

        return $comment->fresh();
    }

    public function deleteComment(Comment $comment, Employee $actor): void
    {
        $this->ensureCommentManage($comment, $actor);
        $comment->delete();
    }

    public function listDecisions(Project $project, Employee $actor): Collection
    {
        $this->ensureAccess($actor, $project);

        return $project->decisions()->with(['author', 'deciders'])->orderByDesc('created_at')->get();
    }

    public function createDecision(Project $project, Employee $actor, array $data): ProjectDecision
    {
        $this->ensureManage($actor, $project);

        $decision = $project->decisions()->create([
            'author_id' => $actor->id,
            'title' => $data['title'],
            'decided_at' => $data['decided_at'] ?? null,
        ]);

        if (! empty($data['decider_ids'])) {
            $decision->deciders()->sync($data['decider_ids']);
        }

        return $decision->load(['author', 'deciders']);
    }

    public function deleteDecision(ProjectDecision $decision, Employee $actor): void
    {
        $this->ensureManage($actor, $decision->project);
        $decision->delete();
    }

    public function listTaskLists(Project $project, Employee $actor): Collection
    {
        $this->ensureAccess($actor, $project);

        return $project->lists()->with(['tasks.assignee', 'tasks.comments'])->orderBy('created_at')->get();
    }

    public function createTaskList(Project $project, Employee $actor, array $data): ProjectTaskList
    {
        $this->ensureManage($actor, $project);

        return $project->lists()->create([
            'author_id' => $actor->id,
            'title' => $data['title'],
            'description' => $data['description'] ?? null,
        ])->load(['tasks.assignee', 'tasks.comments']);
    }

    public function updateTaskList(ProjectTaskList $list, Employee $actor, array $data): ProjectTaskList
    {
        $this->ensureManage($actor, $list->project);
        $list->update([
            'title' => $data['title'] ?? $list->title,
            'description' => array_key_exists('description', $data) ? $data['description'] : $list->description,
        ]);

        return $list->fresh(['tasks.assignee', 'tasks.comments']);
    }

    public function deleteTaskList(ProjectTaskList $list, Employee $actor): void
    {
        $this->ensureManage($actor, $list->project);
        $list->delete();
    }

    public function findTask(Project $project, string $taskId): ProjectTask
    {
        return $project->tasks()->with(['assignee', 'comments'])->where('id', $taskId)->firstOrFail();
    }

    public function createTask(Project $project, Employee $actor, array $data): ProjectTask
    {
        $this->ensureManage($actor, $project);

        if (! empty($data['project_task_list_id'])) {
            ProjectTaskList::query()
                ->where('project_id', $project->id)
                ->where('id', $data['project_task_list_id'])
                ->firstOrFail();
        }

        if (! empty($data['assignee_id'])) {
            Employee::query()
                ->where('company_id', $project->company_id)
                ->where('id', $data['assignee_id'])
                ->firstOrFail();
        }

        return $project->tasks()->create([
            'project_task_list_id' => $data['project_task_list_id'] ?? null,
            'author_id' => $actor->id,
            'assignee_id' => $data['assignee_id'] ?? null,
            'title' => $data['title'],
            'description' => $data['description'] ?? null,
        ])->load(['assignee', 'comments']);
    }

    public function updateTask(ProjectTask $task, Employee $actor, array $data): ProjectTask
    {
        $this->ensureManage($actor, $task->project);

        if (! empty($data['project_task_list_id'])) {
            ProjectTaskList::query()
                ->where('project_id', $task->project_id)
                ->where('id', $data['project_task_list_id'])
                ->firstOrFail();
        }

        if (! empty($data['assignee_id'])) {
            Employee::query()
                ->where('company_id', $task->project->company_id)
                ->where('id', $data['assignee_id'])
                ->firstOrFail();
        }

        $task->update([
            'title' => $data['title'] ?? $task->title,
            'description' => array_key_exists('description', $data) ? $data['description'] : $task->description,
            'project_task_list_id' => array_key_exists('project_task_list_id', $data)
                ? $data['project_task_list_id']
                : $task->project_task_list_id,
            'assignee_id' => array_key_exists('assignee_id', $data) ? $data['assignee_id'] : $task->assignee_id,
        ]);

        return $task->fresh(['assignee', 'comments']);
    }

    public function deleteTask(ProjectTask $task, Employee $actor): void
    {
        $this->ensureManage($actor, $task->project);
        $task->delete();
    }

    public function toggleTask(ProjectTask $task, Employee $actor): ProjectTask
    {
        $this->ensureManage($actor, $task->project);

        $completed = ! $task->completed;
        $task->update([
            'completed' => $completed,
            'completed_at' => $completed ? now() : null,
        ]);

        return $task->fresh(['assignee', 'comments']);
    }

    public function listTaskTimeEntries(ProjectTask $task, Employee $actor): Collection
    {
        $this->ensureAccess($actor, $task->project);

        return TimeTrackingEntry::query()
            ->where('project_task_id', $task->id)
            ->orderByDesc('happened_at')
            ->get();
    }

    public function createTaskComment(Company $company, ProjectTask $task, Employee $actor, array $data): Comment
    {
        $this->ensureManage($actor, $task->project);

        return $task->comments()->create([
            'company_id' => $company->id,
            'author_id' => $actor->id,
            'author_name' => $actor->fullName(),
            'content' => $data['content'],
        ]);
    }

    public function listBoards(Project $project, Employee $actor): Collection
    {
        $this->ensureAccess($actor, $project);

        return $project->boards()->orderBy('name')->get();
    }

    public function createBoard(Project $project, Employee $actor, string $name): ProjectBoard
    {
        $this->ensureManage($actor, $project);

        return DB::transaction(function () use ($project, $name) {
            $board = $project->boards()->create(['name' => $name]);

            $board->sprints()->create([
                'project_id' => $project->id,
                'name' => 'Backlog',
                'active' => false,
                'position' => 0,
            ]);

            $board->sprints()->create([
                'project_id' => $project->id,
                'name' => 'Sprint 1',
                'active' => false,
                'position' => 1,
            ]);

            return $board;
        });
    }

    public function getBoard(ProjectBoard $board, Employee $actor): ProjectBoard
    {
        $this->ensureAccess($actor, $board->project);

        return $board->load([
            'sprints' => fn ($query) => $query->orderBy('position'),
            'sprints.issues.type',
            'sprints.issues.assignees',
        ]);
    }

    public function updateBoard(ProjectBoard $board, Employee $actor, string $name): ProjectBoard
    {
        $this->ensureManage($actor, $board->project);
        $board->update(['name' => $name]);

        return $board;
    }

    public function deleteBoard(ProjectBoard $board, Employee $actor): void
    {
        $this->ensureManage($actor, $board->project);
        $board->delete();
    }

    public function getBacklog(ProjectBoard $board, Employee $actor): ProjectSprint
    {
        $this->ensureAccess($actor, $board->project);

        return $this->findOrCreateBacklog($board)->load([
            'issues.type',
            'issues.assignees',
        ]);
    }

    public function startSprint(ProjectSprint $sprint, Employee $actor): ProjectSprint
    {
        $this->ensureManage($actor, $sprint->project);

        if ($sprint->name === 'Backlog') {
            throw new RuntimeException('Backlog sprint cannot be started', 409);
        }

        $sprint->update([
            'active' => true,
            'started_at' => now(),
            'completed_at' => null,
        ]);

        return $sprint->fresh(['issues.type', 'issues.assignees']);
    }

    public function toggleSprint(ProjectSprint $sprint, Employee $actor): ProjectSprint
    {
        $this->ensureManage($actor, $sprint->project);

        if ($sprint->name === 'Backlog') {
            throw new RuntimeException('Backlog sprint cannot be toggled', 409);
        }

        if ($sprint->completed_at !== null) {
            $sprint->update([
                'active' => false,
                'completed_at' => null,
            ]);
        } else {
            $sprint->update([
                'active' => false,
                'completed_at' => now(),
            ]);
        }

        return $sprint->fresh(['issues.type', 'issues.assignees']);
    }

    public function createIssue(
        Project $project,
        ProjectBoard $board,
        ProjectSprint $sprint,
        Employee $actor,
        array $data,
    ): ProjectIssue {
        $this->ensureManage($actor, $project);

        if ((string) $board->project_id !== (string) $project->id
            || (string) $sprint->project_id !== (string) $project->id) {
            throw new RuntimeException('Board or sprint does not belong to project', 422);
        }

        if (! empty($data['issue_type_id'])) {
            IssueType::query()
                ->where('company_id', $project->company_id)
                ->where('id', $data['issue_type_id'])
                ->firstOrFail();
        }

        return DB::transaction(function () use ($project, $board, $sprint, $actor, $data) {
            $idInProject = (int) ProjectIssue::query()->where('project_id', $project->id)->max('id_in_project') + 1;
            $shortCode = $project->short_code ?? 'PRJ';
            $key = $shortCode.'-'.$idInProject;

            $issue = ProjectIssue::query()->create([
                'project_id' => $project->id,
                'project_board_id' => $board->id,
                'reporter_id' => $actor->id,
                'issue_type_id' => $data['issue_type_id'] ?? null,
                'is_separator' => (bool) ($data['is_separator'] ?? false),
                'id_in_project' => $idInProject,
                'key' => $key,
                'slug' => Str::slug($data['title']),
                'title' => $data['title'],
                'description' => $data['description'] ?? null,
                'story_points' => $data['story_points'] ?? null,
            ]);

            $maxPosition = (int) DB::table('project_issue_project_sprint')
                ->where('project_sprint_id', $sprint->id)
                ->max('position');

            $issue->sprints()->attach($sprint->id, ['position' => $maxPosition + 1]);

            return $issue->load(['type', 'assignees']);
        });
    }

    public function reorderIssue(
        ProjectSprint $sprint,
        ProjectIssue $issue,
        Employee $actor,
        int $position,
    ): ProjectIssue {
        $this->ensureManage($actor, $issue->project);

        $pivot = DB::table('project_issue_project_sprint')
            ->where('project_sprint_id', $sprint->id)
            ->where('project_issue_id', $issue->id)
            ->first();

        if ($pivot === null) {
            throw new RuntimeException('Issue is not in this sprint', 404);
        }

        $pastPosition = (int) $pivot->position;

        if ($position > $pastPosition) {
            DB::table('project_issue_project_sprint')
                ->where('project_sprint_id', $sprint->id)
                ->where('position', '>', $pastPosition)
                ->where('position', '<=', $position)
                ->decrement('position');
        } else {
            DB::table('project_issue_project_sprint')
                ->where('project_sprint_id', $sprint->id)
                ->where('position', '>=', $position)
                ->where('position', '<', $pastPosition)
                ->increment('position');
        }

        DB::table('project_issue_project_sprint')
            ->where('project_sprint_id', $sprint->id)
            ->where('project_issue_id', $issue->id)
            ->update(['position' => $position]);

        return $issue->fresh(['type', 'assignees']);
    }

    public function deleteIssue(ProjectIssue $issue, Employee $actor): void
    {
        $this->ensureManage($actor, $issue->project);
        $issue->delete();
    }

    public function addIssueAssignee(ProjectIssue $issue, Employee $actor, Employee $assignee): ProjectIssue
    {
        $this->ensureManage($actor, $issue->project);

        if ((string) $assignee->company_id !== (string) $issue->project->company_id) {
            throw new RuntimeException('Employee does not belong to this company', 422);
        }

        $issue->assignees()->syncWithoutDetaching([(string) $assignee->id]);

        return $issue->fresh(['type', 'assignees']);
    }

    public function removeIssueAssignee(ProjectIssue $issue, Employee $actor, Employee $assignee): ProjectIssue
    {
        $this->ensureManage($actor, $issue->project);
        $issue->assignees()->detach($assignee->id);

        return $issue->fresh(['type', 'assignees']);
    }

    public function setIssuePoints(ProjectIssue $issue, Employee $actor, ?int $points): ProjectIssue
    {
        $this->ensureManage($actor, $issue->project);
        $issue->update(['story_points' => $points]);

        return $issue->fresh(['type', 'assignees']);
    }

    public function listProjectsForTimesheet(Company $company, Employee $employee): Collection
    {
        return Project::query()
            ->where('company_id', $company->id)
            ->whereHas('employees', fn (Builder $q) => $q->where('employees.id', $employee->id))
            ->orderBy('name')
            ->get();
    }

    public function listTasksForTimesheet(Company $company, Employee $employee, string $projectId): Collection
    {
        $project = Project::query()
            ->where('company_id', $company->id)
            ->where('id', $projectId)
            ->firstOrFail();

        if (! $this->isMember($employee, $project)) {
            throw new RuntimeException('Forbidden', 403);
        }

        return ProjectTask::query()
            ->where('project_id', $project->id)
            ->orderBy('title')
            ->get(['id', 'title', 'completed', 'project_task_list_id']);
    }

    public function projectSummaryPayload(Project $project): array
    {
        return [
            'id' => (string) $project->id,
            'name' => $project->name,
            'code' => $project->code,
            'short_code' => $project->short_code,
            'status' => $project->status,
            'emoji' => $project->emoji,
        ];
    }

    public function projectPayload(Project $project): array
    {
        $project->loadMissing(['lead', 'employees', 'teams']);

        return [
            'id' => (string) $project->id,
            'company_id' => (string) $project->company_id,
            'name' => $project->name,
            'code' => $project->code,
            'short_code' => $project->short_code,
            'emoji' => $project->emoji,
            'summary' => $project->summary,
            'description' => $project->description,
            'status' => $project->status,
            'completed' => (bool) $project->completed,
            'project_lead_id' => $project->project_lead_id ? (string) $project->project_lead_id : null,
            'lead' => $project->lead ? $this->employeeSummary($project->lead) : null,
            'started_at' => $project->started_at?->toIso8601String(),
            'planned_finished_at' => $project->planned_finished_at?->toIso8601String(),
            'actually_finished_at' => $project->actually_finished_at?->toIso8601String(),
            'members' => $project->employees->map(fn (Employee $employee) => $this->employeeSummary($employee))->values()->all(),
            'teams' => $project->teams->map(fn (Team $team) => [
                'id' => (string) $team->id,
                'name' => $team->name,
            ])->values()->all(),
            'member_count' => $project->employees->count(),
        ];
    }

    public function linkPayload(ProjectLink $link): array
    {
        return [
            'id' => (string) $link->id,
            'project_id' => (string) $link->project_id,
            'type' => $link->type,
            'label' => $link->label,
            'url' => $link->url,
        ];
    }

    public function statusPayload(ProjectStatus $status): array
    {
        $status->loadMissing('author');

        return [
            'id' => (string) $status->id,
            'project_id' => (string) $status->project_id,
            'author_id' => $status->author_id ? (string) $status->author_id : null,
            'author_name' => $status->author?->fullName(),
            'title' => $status->title,
            'status' => $status->status,
            'description' => $status->description,
            'created_at' => $status->created_at?->toIso8601String(),
        ];
    }

    public function filePayload(Media $media): array
    {
        return [
            'id' => (int) $media->id,
            'file_name' => $media->file_name,
            'mime_type' => $media->mime_type,
            'size' => (int) $media->size,
            'url' => $media->getUrl(),
        ];
    }

    public function commentPayload(Comment $comment): array
    {
        return [
            'id' => (string) $comment->id,
            'company_id' => (string) $comment->company_id,
            'author_id' => $comment->author_id ? (string) $comment->author_id : null,
            'author_name' => $comment->author_name,
            'content' => $comment->content,
            'created_at' => $comment->created_at?->toIso8601String(),
        ];
    }

    public function messagePayload(ProjectMessage $message): array
    {
        $message->loadMissing(['author', 'comments']);

        return [
            'id' => (string) $message->id,
            'project_id' => (string) $message->project_id,
            'author_id' => $message->author_id ? (string) $message->author_id : null,
            'author_name' => $message->author?->fullName(),
            'title' => $message->title,
            'content' => $message->content,
            'comments' => $message->comments->map(fn (Comment $comment) => $this->commentPayload($comment))->values()->all(),
            'created_at' => $message->created_at?->toIso8601String(),
        ];
    }

    public function decisionPayload(ProjectDecision $decision): array
    {
        $decision->loadMissing(['author', 'deciders']);

        return [
            'id' => (string) $decision->id,
            'project_id' => (string) $decision->project_id,
            'author_id' => $decision->author_id ? (string) $decision->author_id : null,
            'author_name' => $decision->author?->fullName(),
            'title' => $decision->title,
            'decided_at' => $decision->decided_at?->toDateString(),
            'deciders' => $decision->deciders->map(fn (Employee $employee) => $this->employeeSummary($employee))->values()->all(),
        ];
    }

    public function taskListPayload(ProjectTaskList $list): array
    {
        $list->loadMissing(['tasks.assignee', 'tasks.comments']);

        return [
            'id' => (string) $list->id,
            'project_id' => (string) $list->project_id,
            'author_id' => $list->author_id ? (string) $list->author_id : null,
            'title' => $list->title,
            'description' => $list->description,
            'tasks' => $list->tasks->map(fn (ProjectTask $task) => $this->taskPayload($task))->values()->all(),
        ];
    }

    public function taskPayload(ProjectTask $task): array
    {
        $task->loadMissing(['assignee', 'comments']);

        return [
            'id' => (string) $task->id,
            'project_id' => (string) $task->project_id,
            'project_task_list_id' => $task->project_task_list_id ? (string) $task->project_task_list_id : null,
            'author_id' => $task->author_id ? (string) $task->author_id : null,
            'assignee_id' => $task->assignee_id ? (string) $task->assignee_id : null,
            'assignee' => $task->assignee ? $this->employeeSummary($task->assignee) : null,
            'title' => $task->title,
            'description' => $task->description,
            'completed' => (bool) $task->completed,
            'completed_at' => $task->completed_at?->toIso8601String(),
            'comments' => $task->comments->map(fn (Comment $comment) => $this->commentPayload($comment))->values()->all(),
        ];
    }

    public function taskSummaryPayload(ProjectTask $task): array
    {
        return [
            'id' => (string) $task->id,
            'title' => $task->title,
            'completed' => (bool) $task->completed,
            'project_task_list_id' => $task->project_task_list_id ? (string) $task->project_task_list_id : null,
        ];
    }

    public function boardPayload(ProjectBoard $board, bool $detailed = false): array
    {
        $payload = [
            'id' => (string) $board->id,
            'project_id' => (string) $board->project_id,
            'name' => $board->name,
        ];

        if ($detailed) {
            $board->loadMissing(['sprints.issues.type', 'sprints.issues.assignees']);
            $payload['sprints'] = $board->sprints
                ->sortBy('position')
                ->values()
                ->map(fn (ProjectSprint $sprint) => $this->sprintPayload($sprint, true))
                ->all();
        }

        return $payload;
    }

    public function sprintPayload(ProjectSprint $sprint, bool $withIssues = false): array
    {
        $payload = [
            'id' => (string) $sprint->id,
            'project_id' => (string) $sprint->project_id,
            'project_board_id' => $sprint->project_board_id ? (string) $sprint->project_board_id : null,
            'name' => $sprint->name,
            'active' => (bool) $sprint->active,
            'position' => $sprint->position,
            'started_at' => $sprint->started_at?->toIso8601String(),
            'completed_at' => $sprint->completed_at?->toIso8601String(),
        ];

        if ($withIssues) {
            $sprint->loadMissing(['issues.type', 'issues.assignees']);
            $payload['issues'] = $sprint->issues->map(fn (ProjectIssue $issue) => $this->issuePayload($issue))->values()->all();
        }

        return $payload;
    }

    public function issuePayload(ProjectIssue $issue): array
    {
        $issue->loadMissing(['type', 'assignees']);

        return [
            'id' => (string) $issue->id,
            'project_id' => (string) $issue->project_id,
            'project_board_id' => $issue->project_board_id ? (string) $issue->project_board_id : null,
            'reporter_id' => $issue->reporter_id ? (string) $issue->reporter_id : null,
            'issue_type_id' => $issue->issue_type_id ? (string) $issue->issue_type_id : null,
            'issue_type' => $issue->type ? $this->issueTypePayload($issue->type) : null,
            'is_separator' => (bool) $issue->is_separator,
            'id_in_project' => (int) $issue->id_in_project,
            'key' => $issue->key,
            'slug' => $issue->slug,
            'title' => $issue->title,
            'description' => $issue->description,
            'story_points' => $issue->story_points,
            'position' => $issue->pivot?->position,
            'assignees' => $issue->assignees->map(fn (Employee $employee) => $this->employeeSummary($employee))->values()->all(),
        ];
    }

    public function issueTypePayload(IssueType $type): array
    {
        return [
            'id' => (string) $type->id,
            'company_id' => (string) $type->company_id,
            'name' => $type->name,
            'icon' => $type->icon,
        ];
    }

    public function employeeSummary(Employee $employee): array
    {
        return [
            'id' => (string) $employee->id,
            'first_name' => $employee->first_name,
            'last_name' => $employee->last_name,
            'email' => $employee->email,
        ];
    }

    public function timeEntryPayload(TimeTrackingEntry $entry): array
    {
        return [
            'id' => (string) $entry->id,
            'timesheet_id' => (string) $entry->timesheet_id,
            'employee_id' => (string) $entry->employee_id,
            'duration' => (int) $entry->duration,
            'happened_at' => $entry->happened_at->toDateString(),
            'description' => $entry->description,
        ];
    }

    private function isMember(Employee $employee, Project $project): bool
    {
        return $project->employees()->where('employees.id', $employee->id)->exists();
    }

    private function ensureAccess(Employee $employee, Project $project): void
    {
        if (! $this->canAccess($employee, $project)) {
            throw new RuntimeException('Forbidden', 403);
        }
    }

    private function ensureManage(Employee $employee, Project $project): void
    {
        if (! $this->canManage($employee, $project)) {
            throw new RuntimeException('Forbidden', 403);
        }
    }

    private function ensureCommentManage(Comment $comment, Employee $actor): void
    {
        $comment->loadMissing('commentable');
        $commentable = $comment->commentable;
        if ($commentable instanceof ProjectMessage) {
            $this->ensureManage($actor, $commentable->project);

            return;
        }

        if ($commentable instanceof ProjectTask) {
            $this->ensureManage($actor, $commentable->project);

            return;
        }

        throw new RuntimeException('Forbidden', 403);
    }

    private function findOrCreateBacklog(ProjectBoard $board): ProjectSprint
    {
        $backlog = ProjectSprint::query()
            ->where('project_board_id', $board->id)
            ->where('name', 'Backlog')
            ->first();

        if ($backlog !== null) {
            return $backlog;
        }

        return ProjectSprint::query()->create([
            'project_id' => $board->project_id,
            'project_board_id' => $board->id,
            'name' => 'Backlog',
            'active' => false,
            'position' => 0,
        ]);
    }

    private function generateShortCode(Company $company, string $name): string
    {
        $words = preg_split('/\s+/', trim($name)) ?: [];
        $code = '';

        foreach ($words as $word) {
            if ($word === '') {
                continue;
            }

            $code .= strtoupper(substr($word, 0, 1));

            if (strlen($code) >= 3) {
                break;
            }
        }

        if (strlen($code) < 3) {
            $alpha = strtoupper(preg_replace('/[^A-Za-z0-9]/', '', Str::slug($name, '')) ?? '');
            $code = str_pad($code, 3, $alpha !== '' ? $alpha : 'PRJ');
            $code = substr($code, 0, 3);
        }

        $base = $code;
        $suffix = 1;

        while (Project::query()->where('company_id', $company->id)->where('short_code', $code)->exists()) {
            $code = substr($base, 0, 2).$suffix;
            $suffix++;
        }

        return $code;
    }
}
