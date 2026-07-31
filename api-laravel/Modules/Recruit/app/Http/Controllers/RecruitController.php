<?php

namespace Modules\Recruit\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Recruit\Models\Candidate;
use Modules\Recruit\Models\CandidateStage;
use Modules\Recruit\Models\CandidateStageNote;
use Modules\Recruit\Models\CandidateStageParticipant;
use Modules\Recruit\Models\JobOpening;
use Modules\Recruit\Models\RecruitingStage;
use Modules\Recruit\Models\RecruitingStageTemplate;
use Modules\Recruit\Services\RecruitService;
use RuntimeException;
use Spatie\MediaLibrary\MediaCollections\Models\Media;

class RecruitController extends Controller
{
    public function __construct(private readonly RecruitService $recruit) {}

    public function listTemplates(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return ApiResponse::success($this->recruit->listTemplates($company)->values()->all());
    }

    public function storeTemplate(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $data = $request->validate([
            'name' => ['required', 'string', 'max:255'],
        ]);

        try {
            $this->recruit->ensureManage($actor);
            $template = $this->recruit->createTemplate($company, $data['name']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($template, 'Template created', 201);
    }

    public function showTemplate(Request $request, string $companyId, string $templateId): JsonResponse
    {
        $template = $this->findTemplate($request, $templateId);

        return ApiResponse::success($this->recruit->templatePayload($template->load('stages')));
    }

    public function updateTemplate(Request $request, string $companyId, string $templateId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $template = $this->findTemplate($request, $templateId);
        $data = $request->validate([
            'name' => ['required', 'string', 'max:255'],
        ]);

        try {
            $this->recruit->ensureManage($actor);
            $template = $this->recruit->updateTemplate($template, $data['name']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($template, 'Template updated');
    }

    public function destroyTemplate(Request $request, string $companyId, string $templateId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $template = $this->findTemplate($request, $templateId);

        try {
            $this->recruit->ensureManage($actor);
            $this->recruit->deleteTemplate($template);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Template deleted');
    }

    public function storeStage(Request $request, string $companyId, string $templateId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $template = $this->findTemplate($request, $templateId);
        $data = $request->validate([
            'name' => ['required', 'string', 'max:255'],
        ]);

        try {
            $this->recruit->ensureManage($actor);
            $stage = $this->recruit->createStage($template, $data['name']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($stage, 'Stage created', 201);
    }

    public function updateStage(
        Request $request,
        string $companyId,
        string $templateId,
        string $stageId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $template = $this->findTemplate($request, $templateId);
        $stage = $this->findTemplateStage($template, $stageId);
        $data = $request->validate([
            'name' => ['sometimes', 'string', 'max:255'],
            'position' => ['sometimes', 'integer', 'min:0'],
        ]);

        try {
            $this->recruit->ensureManage($actor);
            $stage = $this->recruit->updateStage($stage, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($stage, 'Stage updated');
    }

    public function destroyStage(
        Request $request,
        string $companyId,
        string $templateId,
        string $stageId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $template = $this->findTemplate($request, $templateId);
        $stage = $this->findTemplateStage($template, $stageId);

        try {
            $this->recruit->ensureManage($actor);
            $this->recruit->deleteStage($stage);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Stage deleted');
    }

    public function listOpenings(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $fulfilled = null;

        if ($request->has('fulfilled')) {
            $fulfilled = filter_var($request->query('fulfilled'), FILTER_VALIDATE_BOOLEAN, FILTER_NULL_ON_FAILURE);
        }

        return ApiResponse::success(
            $this->recruit->listOpenings($company, $actor, $fulfilled)->values()->all(),
        );
    }

    public function storeOpening(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $data = $request->validate([
            'title' => ['required', 'string', 'max:255'],
            'description' => ['required', 'string'],
            'position_id' => ['required', 'uuid'],
            'recruiting_stage_template_id' => ['required', 'uuid'],
            'team_id' => ['nullable', 'uuid'],
            'reference_number' => ['nullable', 'string', 'max:255'],
            'sponsor_ids' => ['nullable', 'array'],
            'sponsor_ids.*' => ['uuid'],
        ]);

        try {
            $this->recruit->ensureManage($actor);
            $opening = $this->recruit->createOpening($company, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($opening, 'Job opening created', 201);
    }

    public function showOpening(Request $request, string $companyId, string $jobOpeningId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);

        try {
            $this->recruit->ensureAccess($actor, $opening);
            $payload = $this->recruit->openingPayload(
                $opening->load(['sponsors', 'position', 'template', 'team']),
            );
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($payload);
    }

    public function updateOpening(Request $request, string $companyId, string $jobOpeningId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $data = $request->validate([
            'title' => ['sometimes', 'string', 'max:255'],
            'description' => ['sometimes', 'string'],
            'position_id' => ['sometimes', 'uuid'],
            'recruiting_stage_template_id' => ['sometimes', 'uuid'],
            'team_id' => ['nullable', 'uuid'],
            'reference_number' => ['nullable', 'string', 'max:255'],
            'sponsor_ids' => ['nullable', 'array'],
            'sponsor_ids.*' => ['uuid'],
        ]);

        try {
            $this->recruit->ensureManage($actor);
            $opening = $this->recruit->updateOpening($opening, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($opening, 'Job opening updated');
    }

    public function destroyOpening(Request $request, string $companyId, string $jobOpeningId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);

        try {
            $this->recruit->ensureManage($actor);
            $this->recruit->deleteOpening($opening);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Job opening deleted');
    }

    public function toggleOpening(Request $request, string $companyId, string $jobOpeningId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);

        try {
            $this->recruit->ensureManage($actor);
            $opening = $this->recruit->toggleOpening($opening);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($opening, 'Job opening toggled');
    }

    public function addSponsor(
        Request $request,
        string $companyId,
        string $jobOpeningId,
        string $employeeId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $sponsor = $this->employee($request, $employeeId);

        try {
            $this->recruit->ensureManage($actor);
            $opening = $this->recruit->addSponsor($opening, $sponsor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($opening, 'Sponsor added');
    }

    public function removeSponsor(
        Request $request,
        string $companyId,
        string $jobOpeningId,
        string $employeeId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $sponsor = $this->employee($request, $employeeId);

        try {
            $this->recruit->ensureManage($actor);
            $opening = $this->recruit->removeSponsor($opening, $sponsor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($opening, 'Sponsor removed');
    }

    public function listCandidates(Request $request, string $companyId, string $jobOpeningId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $data = $request->validate([
            'bucket' => ['nullable', 'string', 'in:to_sort,selected,rejected'],
        ]);
        $bucket = $data['bucket'] ?? 'to_sort';

        try {
            $this->recruit->ensureAccess($actor, $opening);
            $candidates = $this->recruit->listCandidates($opening, $bucket);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($candidates->values()->all());
    }

    public function showCandidate(
        Request $request,
        string $companyId,
        string $jobOpeningId,
        string $candidateId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $candidate = $this->findCandidate($opening, $candidateId);

        try {
            $this->recruit->ensureAccess($actor, $opening);
            $payload = $this->recruit->showCandidate($candidate);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($payload);
    }

    public function processStage(
        Request $request,
        string $companyId,
        string $jobOpeningId,
        string $candidateId,
        string $stageId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $candidate = $this->findCandidate($opening, $candidateId);
        $stage = $this->findCandidateStage($candidate, $stageId);
        $data = $request->validate([
            'accepted' => ['required', 'boolean'],
        ]);

        try {
            $this->recruit->ensureAccess($actor, $opening);
            $payload = $this->recruit->processStage($stage, $actor, (bool) $data['accepted']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($payload, 'Stage processed');
    }

    public function listNotes(
        Request $request,
        string $companyId,
        string $jobOpeningId,
        string $candidateId,
        string $stageId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $candidate = $this->findCandidate($opening, $candidateId);
        $stage = $this->findCandidateStage($candidate, $stageId);

        try {
            $this->recruit->ensureAccess($actor, $opening);
            $notes = $stage->notes()->orderBy('created_at')->get()
                ->map(fn (CandidateStageNote $note) => $this->notePayload($note))
                ->values()
                ->all();
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($notes);
    }

    public function storeNote(
        Request $request,
        string $companyId,
        string $jobOpeningId,
        string $candidateId,
        string $stageId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $candidate = $this->findCandidate($opening, $candidateId);
        $stage = $this->findCandidateStage($candidate, $stageId);
        $data = $request->validate([
            'note' => ['required', 'string'],
        ]);

        try {
            $this->recruit->ensureAccess($actor, $opening);
            $note = $this->recruit->createNote($stage, $actor, $data['note']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($note, 'Note created', 201);
    }

    public function updateNote(
        Request $request,
        string $companyId,
        string $jobOpeningId,
        string $candidateId,
        string $stageId,
        string $noteId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $candidate = $this->findCandidate($opening, $candidateId);
        $stage = $this->findCandidateStage($candidate, $stageId);
        $note = $this->findNote($stage, $noteId);
        $data = $request->validate([
            'note' => ['required', 'string'],
        ]);

        try {
            $this->recruit->ensureAccess($actor, $opening);
            $note = $this->recruit->updateNote($note, $data['note']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($note, 'Note updated');
    }

    public function destroyNote(
        Request $request,
        string $companyId,
        string $jobOpeningId,
        string $candidateId,
        string $stageId,
        string $noteId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $candidate = $this->findCandidate($opening, $candidateId);
        $stage = $this->findCandidateStage($candidate, $stageId);
        $note = $this->findNote($stage, $noteId);

        try {
            $this->recruit->ensureAccess($actor, $opening);
            $this->recruit->deleteNote($note);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Note deleted');
    }

    public function addParticipant(
        Request $request,
        string $companyId,
        string $jobOpeningId,
        string $candidateId,
        string $stageId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $candidate = $this->findCandidate($opening, $candidateId);
        $stage = $this->findCandidateStage($candidate, $stageId);
        $data = $request->validate([
            'employee_id' => ['required', 'uuid'],
        ]);
        $participant = $this->employee($request, $data['employee_id']);

        try {
            $this->recruit->ensureAccess($actor, $opening);
            $row = $this->recruit->addParticipant($stage, $participant);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($row, 'Participant added', 201);
    }

    public function removeParticipant(
        Request $request,
        string $companyId,
        string $jobOpeningId,
        string $candidateId,
        string $stageId,
        string $participantId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $candidate = $this->findCandidate($opening, $candidateId);
        $stage = $this->findCandidateStage($candidate, $stageId);
        $participant = $this->findParticipant($stage, $participantId);

        try {
            $this->recruit->ensureAccess($actor, $opening);
            $this->recruit->removeParticipant($participant);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Participant removed');
    }

    public function hire(
        Request $request,
        string $companyId,
        string $jobOpeningId,
        string $candidateId,
    ): JsonResponse {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $candidate = $this->findCandidate($opening, $candidateId);
        $data = $request->validate([
            'email' => ['required', 'email', 'max:255'],
            'first_name' => ['required', 'string', 'max:255'],
            'last_name' => ['required', 'string', 'max:255'],
            'hired_at' => ['required', 'date'],
        ]);

        try {
            $this->recruit->ensureAccess($actor, $opening);
            $result = $this->recruit->hire($company, $opening, $candidate, $data, $actor);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($result, 'Candidate hired', 201);
    }

    public function listFiles(
        Request $request,
        string $companyId,
        string $jobOpeningId,
        string $candidateId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $candidate = $this->findCandidate($opening, $candidateId);

        try {
            $this->recruit->ensureAccess($actor, $opening);
            $files = $this->recruit->listFiles($candidate);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($files);
    }

    public function attachFile(
        Request $request,
        string $companyId,
        string $jobOpeningId,
        string $candidateId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $candidate = $this->findCandidate($opening, $candidateId);
        $data = $request->validate([
            'temporary_upload_id' => ['required', 'integer', 'exists:temporary_uploads,id'],
            'media_id' => ['required', 'integer', 'exists:media,id'],
        ]);

        try {
            $this->recruit->ensureAccess($actor, $opening);
            $media = $this->recruit->attachFile(
                $candidate,
                (int) $data['temporary_upload_id'],
                (int) $data['media_id'],
            );
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->filePayload($media), 'File attached', 201);
    }

    public function deleteFile(
        Request $request,
        string $companyId,
        string $jobOpeningId,
        string $candidateId,
        int $mediaId,
    ): JsonResponse {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $opening = $this->findOpening($request, $jobOpeningId);
        $candidate = $this->findCandidate($opening, $candidateId);

        try {
            $this->recruit->ensureAccess($actor, $opening);
            $this->recruit->deleteFile($candidate, $mediaId);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'File deleted');
    }

    private function findTemplate(Request $request, string $templateId): RecruitingStageTemplate
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return RecruitingStageTemplate::query()
            ->where('company_id', $company->id)
            ->where('id', $templateId)
            ->firstOrFail();
    }

    private function findTemplateStage(RecruitingStageTemplate $template, string $stageId): RecruitingStage
    {
        return $template->stages()->where('id', $stageId)->firstOrFail();
    }

    private function findOpening(Request $request, string $openingId): JobOpening
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return JobOpening::query()
            ->where('company_id', $company->id)
            ->where('id', $openingId)
            ->firstOrFail();
    }

    private function findCandidate(JobOpening $opening, string $candidateId): Candidate
    {
        return $opening->candidates()->where('id', $candidateId)->firstOrFail();
    }

    private function findCandidateStage(Candidate $candidate, string $stageId): CandidateStage
    {
        return $candidate->stages()->where('id', $stageId)->firstOrFail();
    }

    private function findNote(CandidateStage $stage, string $noteId): CandidateStageNote
    {
        return $stage->notes()->where('id', $noteId)->firstOrFail();
    }

    private function findParticipant(CandidateStage $stage, string $participantId): CandidateStageParticipant
    {
        return $stage->participants()->where('id', $participantId)->firstOrFail();
    }

    private function employee(Request $request, string $id): Employee
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return Employee::query()
            ->where('company_id', $company->id)
            ->where('id', $id)
            ->firstOrFail();
    }

    private function notePayload(CandidateStageNote $note): array
    {
        return [
            'id' => $note->id,
            'author_id' => $note->author_id,
            'author_name' => $note->author_name,
            'note' => $note->note,
            'created_at' => $note->created_at?->toIso8601String(),
        ];
    }

    private function filePayload(Media $media): array
    {
        return [
            'id' => $media->id,
            'file_name' => $media->file_name,
            'mime_type' => $media->mime_type,
            'size' => $media->size,
            'url' => url('/api/v1/media/'.$media->id.'/file'),
        ];
    }
}
