<?php

namespace Modules\Project\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Project\Models\Comment;
use Modules\Project\Models\ProjectBoard;
use Modules\Project\Models\ProjectDecision;
use Modules\Project\Models\ProjectIssue;
use Modules\Project\Models\ProjectLink;
use Modules\Project\Models\ProjectMessage;
use Modules\Project\Models\ProjectSprint;
use Modules\Project\Models\ProjectTask;
use Modules\Project\Models\ProjectTaskList;
use Modules\Project\Services\ProjectService;
use Modules\Team\Models\Team;
use RuntimeException;

class ProjectController extends Controller
{
    public function __construct(private readonly ProjectService $projects) {}

    public function issueTypes(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return ApiResponse::success(
            $this->projects->listIssueTypes($company)
                ->map(fn ($type) => $this->projects->issueTypePayload($type))
                ->values()
                ->all(),
        );
    }

    public function index(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        return ApiResponse::success(
            $this->projects->list($company, $actor)
                ->map(fn ($project) => $this->projects->projectPayload($project))
                ->values()
                ->all(),
        );
    }

    public function store(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $data = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'code' => ['nullable', 'string', 'max:255'],
            'short_code' => ['nullable', 'string', 'max:255'],
            'emoji' => ['nullable', 'string', 'max:16'],
            'summary' => ['nullable', 'string', 'max:255'],
            'description' => ['nullable', 'string'],
            'status' => ['nullable', 'string', 'in:created,started,paused,cancelled,closed'],
            'project_lead_id' => ['nullable', 'uuid'],
            'started_at' => ['nullable', 'date'],
            'planned_finished_at' => ['nullable', 'date'],
        ]);

        try {
            $project = $this->projects->create($company, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->projectPayload($project), 'Project created', 201);
    }

    public function show(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        if (! $this->projects->canAccess($actor, $project)) {
            return ApiResponse::error('Forbidden', 403);
        }

        return ApiResponse::success($this->projects->projectPayload($project));
    }

    public function update(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $data = $request->validate([
            'name' => ['sometimes', 'string', 'max:255'],
            'code' => ['nullable', 'string', 'max:255'],
            'short_code' => ['nullable', 'string', 'max:255'],
            'emoji' => ['nullable', 'string', 'max:16'],
            'summary' => ['nullable', 'string', 'max:255'],
            'description' => ['nullable', 'string'],
            'status' => ['sometimes', 'string', 'in:created,started,paused,cancelled,closed'],
            'project_lead_id' => ['nullable', 'uuid'],
            'started_at' => ['nullable', 'date'],
            'planned_finished_at' => ['nullable', 'date'],
        ]);

        try {
            $project = $this->projects->update($project, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->projectPayload($project), 'Project updated');
    }

    public function destroy(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        try {
            $this->projects->delete($project, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Project deleted');
    }

    public function addMember(Request $request, string $companyId, string $projectId, string $employeeId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $member = $this->employee($request, $employeeId);
        $data = $request->validate(['role' => ['nullable', 'string', 'max:255']]);

        try {
            $project = $this->projects->addMember($project, $actor, $member, $data['role'] ?? null);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->projectPayload($project), 'Member added');
    }

    public function removeMember(Request $request, string $companyId, string $projectId, string $employeeId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $member = $this->employee($request, $employeeId);

        try {
            $project = $this->projects->removeMember($project, $actor, $member);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->projectPayload($project), 'Member removed');
    }

    public function setLead(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $data = $request->validate(['employee_id' => ['nullable', 'uuid']]);
        $lead = isset($data['employee_id']) ? $this->employee($request, $data['employee_id']) : null;

        try {
            $project = $this->projects->setLead($project, $actor, $lead);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->projectPayload($project), 'Lead updated');
    }

    public function attachTeam(Request $request, string $companyId, string $projectId, string $teamId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $team = $this->team($request, $teamId);

        try {
            $project = $this->projects->attachTeam($project, $actor, $team);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->projectPayload($project), 'Team attached');
    }

    public function detachTeam(Request $request, string $companyId, string $projectId, string $teamId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $team = $this->team($request, $teamId);

        try {
            $project = $this->projects->detachTeam($project, $actor, $team);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->projectPayload($project), 'Team detached');
    }

    public function links(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        try {
            $items = $this->projects->listLinks($project, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(
            $items->map(fn (ProjectLink $link) => $this->projects->linkPayload($link))->values()->all(),
        );
    }

    public function createLink(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $data = $request->validate([
            'type' => ['required', 'string', 'max:255'],
            'label' => ['nullable', 'string', 'max:255'],
            'url' => ['required', 'string', 'max:2048'],
        ]);

        try {
            $link = $this->projects->createLink($project, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->linkPayload($link), 'Link created', 201);
    }

    public function updateLink(Request $request, string $companyId, string $projectId, string $linkId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $link = $this->link($request, $projectId, $linkId);
        $data = $request->validate([
            'type' => ['sometimes', 'string', 'max:255'],
            'label' => ['nullable', 'string', 'max:255'],
            'url' => ['sometimes', 'string', 'max:2048'],
        ]);

        try {
            $link = $this->projects->updateLink($link, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->linkPayload($link), 'Link updated');
    }

    public function deleteLink(Request $request, string $companyId, string $projectId, string $linkId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $link = $this->link($request, $projectId, $linkId);

        try {
            $this->projects->deleteLink($link, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Link deleted');
    }

    public function statuses(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        try {
            $items = $this->projects->listStatuses($project, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(
            $items->map(fn ($status) => $this->projects->statusPayload($status))->values()->all(),
        );
    }

    public function createStatus(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $data = $request->validate([
            'title' => ['required', 'string', 'max:255'],
            'status' => ['required', 'string', 'max:255'],
            'description' => ['required', 'string'],
        ]);

        try {
            $status = $this->projects->createStatus($project, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->statusPayload($status), 'Status created', 201);
    }

    public function files(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        try {
            $items = $this->projects->listFiles($project, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(
            $items->map(fn ($media) => $this->projects->filePayload($media))->values()->all(),
        );
    }

    public function attachFile(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $data = $request->validate([
            'temporary_upload_id' => ['required', 'integer', 'exists:temporary_uploads,id'],
            'media_id' => ['required', 'integer', 'exists:media,id'],
        ]);

        try {
            $media = $this->projects->attachFile(
                $project,
                $actor,
                (int) $data['temporary_upload_id'],
                (int) $data['media_id'],
            );
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->filePayload($media), 'File attached', 201);
    }

    public function deleteFile(Request $request, string $companyId, string $projectId, int $mediaId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        try {
            $this->projects->deleteFile($project, $actor, $mediaId);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'File deleted');
    }

    public function messages(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        try {
            $items = $this->projects->listMessages($project, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(
            $items->map(fn (ProjectMessage $message) => $this->projects->messagePayload($message))->values()->all(),
        );
    }

    public function createMessage(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $data = $request->validate([
            'title' => ['required', 'string', 'max:255'],
            'content' => ['required', 'string'],
        ]);

        try {
            $message = $this->projects->createMessage($project, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->messagePayload($message), 'Message created', 201);
    }

    public function showMessage(Request $request, string $companyId, string $projectId, string $messageId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        try {
            $this->projects->listMessages($project, $actor);
            $message = $this->projects->findMessage($project, $messageId);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->messagePayload($message));
    }

    public function updateMessage(Request $request, string $companyId, string $projectId, string $messageId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $message = $this->message($request, $projectId, $messageId);
        $data = $request->validate([
            'title' => ['sometimes', 'string', 'max:255'],
            'content' => ['sometimes', 'string'],
        ]);

        try {
            $message = $this->projects->updateMessage($message, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->messagePayload($message), 'Message updated');
    }

    public function deleteMessage(Request $request, string $companyId, string $projectId, string $messageId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $message = $this->message($request, $projectId, $messageId);

        try {
            $this->projects->deleteMessage($message, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Message deleted');
    }

    public function createMessageComment(Request $request, string $companyId, string $projectId, string $messageId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $message = $this->message($request, $projectId, $messageId);
        $data = $request->validate(['content' => ['required', 'string']]);

        try {
            $comment = $this->projects->createMessageComment($company, $message, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->commentPayload($comment), 'Comment created', 201);
    }

    public function updateMessageComment(
        Request $request,
        string $companyId,
        string $projectId,
        string $messageId,
        string $commentId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $comment = $this->comment($commentId);
        $data = $request->validate(['content' => ['required', 'string']]);

        try {
            $comment = $this->projects->updateComment($comment, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->commentPayload($comment), 'Comment updated');
    }

    public function deleteMessageComment(
        Request $request,
        string $companyId,
        string $projectId,
        string $messageId,
        string $commentId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $comment = $this->comment($commentId);

        try {
            $this->projects->deleteComment($comment, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Comment deleted');
    }

    public function decisions(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        try {
            $items = $this->projects->listDecisions($project, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(
            $items->map(fn (ProjectDecision $decision) => $this->projects->decisionPayload($decision))->values()->all(),
        );
    }

    public function createDecision(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $data = $request->validate([
            'title' => ['required', 'string', 'max:255'],
            'decided_at' => ['nullable', 'date'],
            'decider_ids' => ['nullable', 'array'],
            'decider_ids.*' => ['uuid'],
        ]);

        try {
            $decision = $this->projects->createDecision($project, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->decisionPayload($decision), 'Decision created', 201);
    }

    public function deleteDecision(Request $request, string $companyId, string $projectId, string $decisionId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $decision = $this->decision($request, $projectId, $decisionId);

        try {
            $this->projects->deleteDecision($decision, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Decision deleted');
    }

    public function taskLists(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        try {
            $items = $this->projects->listTaskLists($project, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(
            $items->map(fn (ProjectTaskList $list) => $this->projects->taskListPayload($list))->values()->all(),
        );
    }

    public function createTaskList(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $data = $request->validate([
            'title' => ['required', 'string', 'max:255'],
            'description' => ['nullable', 'string'],
        ]);

        try {
            $list = $this->projects->createTaskList($project, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->taskListPayload($list), 'Task list created', 201);
    }

    public function updateTaskList(Request $request, string $companyId, string $projectId, string $listId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $list = $this->taskList($request, $projectId, $listId);
        $data = $request->validate([
            'title' => ['sometimes', 'string', 'max:255'],
            'description' => ['nullable', 'string'],
        ]);

        try {
            $list = $this->projects->updateTaskList($list, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->taskListPayload($list), 'Task list updated');
    }

    public function deleteTaskList(Request $request, string $companyId, string $projectId, string $listId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $list = $this->taskList($request, $projectId, $listId);

        try {
            $this->projects->deleteTaskList($list, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Task list deleted');
    }

    public function createTask(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $data = $request->validate([
            'title' => ['required', 'string', 'max:255'],
            'description' => ['nullable', 'string'],
            'project_task_list_id' => ['nullable', 'uuid'],
            'assignee_id' => ['nullable', 'uuid'],
        ]);

        try {
            $task = $this->projects->createTask($project, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->taskPayload($task), 'Task created', 201);
    }

    public function showTask(Request $request, string $companyId, string $projectId, string $taskId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        try {
            $this->projects->listTaskLists($project, $actor);
            $task = $this->projects->findTask($project, $taskId);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->taskPayload($task));
    }

    public function updateTask(Request $request, string $companyId, string $projectId, string $taskId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $task = $this->task($request, $projectId, $taskId);
        $data = $request->validate([
            'title' => ['sometimes', 'string', 'max:255'],
            'description' => ['nullable', 'string'],
            'project_task_list_id' => ['nullable', 'uuid'],
            'assignee_id' => ['nullable', 'uuid'],
        ]);

        try {
            $task = $this->projects->updateTask($task, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->taskPayload($task), 'Task updated');
    }

    public function deleteTask(Request $request, string $companyId, string $projectId, string $taskId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $task = $this->task($request, $projectId, $taskId);

        try {
            $this->projects->deleteTask($task, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Task deleted');
    }

    public function toggleTask(Request $request, string $companyId, string $projectId, string $taskId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $task = $this->task($request, $projectId, $taskId);

        try {
            $task = $this->projects->toggleTask($task, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->taskPayload($task), 'Task toggled');
    }

    public function taskTimeEntries(Request $request, string $companyId, string $projectId, string $taskId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $task = $this->task($request, $projectId, $taskId);

        try {
            $entries = $this->projects->listTaskTimeEntries($task, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(
            $entries->map(fn ($entry) => $this->projects->timeEntryPayload($entry))->values()->all(),
        );
    }

    public function createTaskComment(Request $request, string $companyId, string $projectId, string $taskId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $task = $this->task($request, $projectId, $taskId);
        $data = $request->validate(['content' => ['required', 'string']]);

        try {
            $comment = $this->projects->createTaskComment($company, $task, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->commentPayload($comment), 'Comment created', 201);
    }

    public function updateTaskComment(
        Request $request,
        string $companyId,
        string $projectId,
        string $taskId,
        string $commentId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $comment = $this->comment($commentId);
        $data = $request->validate(['content' => ['required', 'string']]);

        try {
            $comment = $this->projects->updateComment($comment, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->commentPayload($comment), 'Comment updated');
    }

    public function deleteTaskComment(
        Request $request,
        string $companyId,
        string $projectId,
        string $taskId,
        string $commentId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $comment = $this->comment($commentId);

        try {
            $this->projects->deleteComment($comment, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Comment deleted');
    }

    public function boards(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        try {
            $items = $this->projects->listBoards($project, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(
            $items->map(fn (ProjectBoard $board) => $this->projects->boardPayload($board))->values()->all(),
        );
    }

    public function createBoard(Request $request, string $companyId, string $projectId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $data = $request->validate(['name' => ['required', 'string', 'max:255']]);

        try {
            $board = $this->projects->createBoard($project, $actor, $data['name']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->boardPayload($board), 'Board created', 201);
    }

    public function showBoard(Request $request, string $companyId, string $projectId, string $boardId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $board = $this->board($request, $projectId, $boardId);

        try {
            $board = $this->projects->getBoard($board, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->boardPayload($board, detailed: true));
    }

    public function updateBoard(Request $request, string $companyId, string $projectId, string $boardId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $board = $this->board($request, $projectId, $boardId);
        $data = $request->validate(['name' => ['required', 'string', 'max:255']]);

        try {
            $board = $this->projects->updateBoard($board, $actor, $data['name']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->boardPayload($board), 'Board updated');
    }

    public function deleteBoard(Request $request, string $companyId, string $projectId, string $boardId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $board = $this->board($request, $projectId, $boardId);

        try {
            $this->projects->deleteBoard($board, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Board deleted');
    }

    public function backlog(Request $request, string $companyId, string $projectId, string $boardId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $board = $this->board($request, $projectId, $boardId);

        try {
            $sprint = $this->projects->getBacklog($board, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->sprintPayload($sprint, withIssues: true));
    }

    public function startSprint(
        Request $request,
        string $companyId,
        string $projectId,
        string $boardId,
        string $sprintId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $sprint = $this->sprint($request, $projectId, $boardId, $sprintId);

        try {
            $sprint = $this->projects->startSprint($sprint, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->sprintPayload($sprint, withIssues: true), 'Sprint started');
    }

    public function toggleSprint(
        Request $request,
        string $companyId,
        string $projectId,
        string $boardId,
        string $sprintId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $sprint = $this->sprint($request, $projectId, $boardId, $sprintId);

        try {
            $sprint = $this->projects->toggleSprint($sprint, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->sprintPayload($sprint, withIssues: true), 'Sprint toggled');
    }

    public function createIssue(
        Request $request,
        string $companyId,
        string $projectId,
        string $boardId,
        string $sprintId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $project = $this->projects->find($request->attributes->get('company'), $projectId);
        $board = $this->board($request, $projectId, $boardId);
        $sprint = $this->sprint($request, $projectId, $boardId, $sprintId);
        $data = $request->validate([
            'title' => ['required', 'string', 'max:255'],
            'description' => ['nullable', 'string'],
            'issue_type_id' => ['nullable', 'uuid'],
            'story_points' => ['nullable', 'integer', 'min:0'],
            'is_separator' => ['nullable', 'boolean'],
        ]);

        try {
            $issue = $this->projects->createIssue($project, $board, $sprint, $actor, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->issuePayload($issue), 'Issue created', 201);
    }

    public function reorderIssue(
        Request $request,
        string $companyId,
        string $projectId,
        string $boardId,
        string $sprintId,
        string $issueId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $sprint = $this->sprint($request, $projectId, $boardId, $sprintId);
        $issue = $this->issue($request, $projectId, $issueId);
        $data = $request->validate(['position' => ['required', 'integer', 'min:0']]);

        try {
            $issue = $this->projects->reorderIssue($sprint, $issue, $actor, (int) $data['position']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->issuePayload($issue), 'Issue reordered');
    }

    public function deleteIssue(
        Request $request,
        string $companyId,
        string $projectId,
        string $boardId,
        string $sprintId,
        string $issueId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $issue = $this->issue($request, $projectId, $issueId);

        try {
            $this->projects->deleteIssue($issue, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Issue deleted');
    }

    public function addIssueAssignee(
        Request $request,
        string $companyId,
        string $projectId,
        string $boardId,
        string $issueId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $issue = $this->issue($request, $projectId, $issueId);
        $data = $request->validate(['employee_id' => ['required', 'uuid']]);
        $assignee = $this->employee($request, $data['employee_id']);

        try {
            $issue = $this->projects->addIssueAssignee($issue, $actor, $assignee);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->issuePayload($issue), 'Assignee added');
    }

    public function removeIssueAssignee(
        Request $request,
        string $companyId,
        string $projectId,
        string $boardId,
        string $issueId,
        string $assigneeId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $issue = $this->issue($request, $projectId, $issueId);
        $assignee = $this->employee($request, $assigneeId);

        try {
            $issue = $this->projects->removeIssueAssignee($issue, $actor, $assignee);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->issuePayload($issue), 'Assignee removed');
    }

    public function setIssuePoints(
        Request $request,
        string $companyId,
        string $projectId,
        string $boardId,
        string $issueId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $issue = $this->issue($request, $projectId, $issueId);
        $data = $request->validate(['story_points' => ['nullable', 'integer', 'min:0']]);

        try {
            $issue = $this->projects->setIssuePoints($issue, $actor, $data['story_points'] ?? null);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->projects->issuePayload($issue), 'Points updated');
    }

    private function employee(Request $request, string $id): Employee
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return Employee::query()->where('company_id', $company->id)->where('id', $id)->firstOrFail();
    }

    private function team(Request $request, string $id): Team
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return Team::query()->where('company_id', $company->id)->where('id', $id)->firstOrFail();
    }

    private function link(Request $request, string $projectId, string $linkId): ProjectLink
    {
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        return $project->links()->where('id', $linkId)->firstOrFail();
    }

    private function message(Request $request, string $projectId, string $messageId): ProjectMessage
    {
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        return $project->messages()->where('id', $messageId)->firstOrFail();
    }

    private function decision(Request $request, string $projectId, string $decisionId): ProjectDecision
    {
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        return $project->decisions()->where('id', $decisionId)->firstOrFail();
    }

    private function taskList(Request $request, string $projectId, string $listId): ProjectTaskList
    {
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        return $project->lists()->where('id', $listId)->firstOrFail();
    }

    private function task(Request $request, string $projectId, string $taskId): ProjectTask
    {
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        return $project->tasks()->where('id', $taskId)->firstOrFail();
    }

    private function board(Request $request, string $projectId, string $boardId): ProjectBoard
    {
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        return $project->boards()->where('id', $boardId)->firstOrFail();
    }

    private function sprint(Request $request, string $projectId, string $boardId, string $sprintId): ProjectSprint
    {
        $board = $this->board($request, $projectId, $boardId);

        return $board->sprints()->where('id', $sprintId)->firstOrFail();
    }

    private function issue(Request $request, string $projectId, string $issueId): ProjectIssue
    {
        $project = $this->projects->find($request->attributes->get('company'), $projectId);

        return $project->issues()->where('id', $issueId)->firstOrFail();
    }

    private function comment(string $commentId): Comment
    {
        return Comment::query()->where('id', $commentId)->firstOrFail();
    }
}
