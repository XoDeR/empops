<?php

namespace Modules\Grow\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Grow\Models\DisciplineCase;
use Modules\Grow\Models\DisciplineEvent;
use Modules\Grow\Models\ECoffeeMatch;
use Modules\Grow\Models\OneOnOneActionItem;
use Modules\Grow\Models\OneOnOneEntry;
use Modules\Grow\Models\OneOnOneNote;
use Modules\Grow\Models\OneOnOneTalkingPoint;
use Modules\Grow\Models\RateYourManagerAnswer;
use Modules\Grow\Models\Skill;
use Modules\Grow\Services\GrowService;
use Modules\Team\Models\Team;
use RuntimeException;

class GrowController extends Controller
{
    public function __construct(private readonly GrowService $grow) {}

    // --- Morale ---

    public function logMorale(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $data = $request->validate([
            'emotion' => ['required', 'integer', 'in:1,2,3'],
            'comment' => ['nullable', 'string', 'max:255'],
        ]);

        try {
            $morale = $this->grow->logMorale($company, $actor, (int) $data['emotion'], $data['comment'] ?? null);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->grow->moralePayload($morale));
    }

    public function todayMorale(Request $request): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $morale = $this->grow->todayMorale($actor);

        return ApiResponse::success($morale ? $this->grow->moralePayload($morale) : null);
    }

    public function companyMoraleHistory(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return ApiResponse::success($this->grow->companyMoraleHistory($company));
    }

    public function teamMoraleHistory(Request $request, string $companyId, string $teamId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $team = Team::query()->where('company_id', $company->id)->where('id', $teamId)->firstOrFail();

        return ApiResponse::success($this->grow->teamMoraleHistory($team));
    }

    public function employeeMorale(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $subject = Employee::query()->where('company_id', $company->id)->where('id', $employeeId)->firstOrFail();

        try {
            return ApiResponse::success($this->grow->employeeMorale($company, $actor, $subject));
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }
    }

    // --- One-on-ones ---

    public function myOneOnOnes(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $this->grow->ensureOpenOneOnOnesForEmployee($company, $actor);

        return ApiResponse::success($this->grow->listOpenOneOnOnesForEmployee($company, $actor));
    }

    public function managerOneOnOnes(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $this->grow->ensureOpenOneOnOnesForManager($company, $actor);

        return ApiResponse::success($this->grow->listOpenOneOnOnesForManager($company, $actor));
    }

    public function showOneOnOne(Request $request, string $companyId, string $entryId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $entry = OneOnOneEntry::query()
            ->where('company_id', $company->id)
            ->where('id', $entryId)
            ->firstOrFail();

        try {
            $this->grow->ensureOneOnOneParticipant($actor, $entry);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->grow->oneOnOnePayload($entry));
    }

    public function markOneOnOneHappened(Request $request, string $companyId, string $entryId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $entry = OneOnOneEntry::query()
            ->where('company_id', $company->id)
            ->where('id', $entryId)
            ->firstOrFail();

        try {
            $this->grow->ensureOneOnOneParticipant($actor, $entry);
            $entry = $this->grow->markOneOnOneHappened($entry);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->grow->oneOnOnePayload($entry));
    }

    public function storeTalkingPoint(Request $request, string $companyId, string $entryId): JsonResponse
    {
        return $this->storeOneOnOneChild($request, $entryId, 'talking_point');
    }

    public function storeActionItem(Request $request, string $companyId, string $entryId): JsonResponse
    {
        return $this->storeOneOnOneChild($request, $entryId, 'action_item');
    }

    public function storeNote(Request $request, string $companyId, string $entryId): JsonResponse
    {
        return $this->storeOneOnOneChild($request, $entryId, 'note');
    }

    private function storeOneOnOneChild(Request $request, string $entryId, string $type): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $entry = OneOnOneEntry::query()
            ->where('company_id', $company->id)
            ->where('id', $entryId)
            ->firstOrFail();

        try {
            $this->grow->ensureOneOnOneParticipant($actor, $entry);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        if ($type === 'note') {
            $data = $request->validate(['note' => ['required', 'string', 'max:65535']]);
            $note = OneOnOneNote::query()->create([
                'one_on_one_entry_id' => $entry->id,
                'note' => $data['note'],
            ]);

            return ApiResponse::success([
                'id' => (string) $note->id,
                'note' => $note->note,
                'created_at' => $note->created_at?->toIso8601String(),
            ]);
        }

        $data = $request->validate(['description' => ['required', 'string', 'max:255']]);
        if ($type === 'talking_point') {
            $item = OneOnOneTalkingPoint::query()->create([
                'one_on_one_entry_id' => $entry->id,
                'description' => $data['description'],
                'checked' => false,
            ]);
        } else {
            $item = OneOnOneActionItem::query()->create([
                'one_on_one_entry_id' => $entry->id,
                'description' => $data['description'],
                'checked' => false,
            ]);
        }

        return ApiResponse::success([
            'id' => (string) $item->id,
            'description' => $item->description,
            'checked' => (bool) $item->checked,
        ]);
    }

    public function toggleTalkingPoint(Request $request, string $companyId, string $entryId, string $pointId): JsonResponse
    {
        return $this->toggleOneOnOneChild($request, $entryId, $pointId, 'talking_point');
    }

    public function toggleActionItem(Request $request, string $companyId, string $entryId, string $itemId): JsonResponse
    {
        return $this->toggleOneOnOneChild($request, $entryId, $itemId, 'action_item');
    }

    private function toggleOneOnOneChild(Request $request, string $entryId, string $childId, string $type): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $entry = OneOnOneEntry::query()
            ->where('company_id', $company->id)
            ->where('id', $entryId)
            ->firstOrFail();

        try {
            $this->grow->ensureOneOnOneParticipant($actor, $entry);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        if ($type === 'talking_point') {
            $item = OneOnOneTalkingPoint::query()
                ->where('one_on_one_entry_id', $entry->id)
                ->where('id', $childId)
                ->firstOrFail();
        } else {
            $item = OneOnOneActionItem::query()
                ->where('one_on_one_entry_id', $entry->id)
                ->where('id', $childId)
                ->firstOrFail();
        }

        $item->checked = ! $item->checked;
        $item->save();

        return ApiResponse::success([
            'id' => (string) $item->id,
            'description' => $item->description,
            'checked' => (bool) $item->checked,
        ]);
    }

    public function destroyTalkingPoint(Request $request, string $companyId, string $entryId, string $pointId): JsonResponse
    {
        return $this->destroyOneOnOneChild($request, $entryId, $pointId, 'talking_point');
    }

    public function destroyActionItem(Request $request, string $companyId, string $entryId, string $itemId): JsonResponse
    {
        return $this->destroyOneOnOneChild($request, $entryId, $itemId, 'action_item');
    }

    public function destroyNote(Request $request, string $companyId, string $entryId, string $noteId): JsonResponse
    {
        return $this->destroyOneOnOneChild($request, $entryId, $noteId, 'note');
    }

    private function destroyOneOnOneChild(Request $request, string $entryId, string $childId, string $type): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $entry = OneOnOneEntry::query()
            ->where('company_id', $company->id)
            ->where('id', $entryId)
            ->firstOrFail();

        try {
            $this->grow->ensureOneOnOneParticipant($actor, $entry);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        match ($type) {
            'talking_point' => OneOnOneTalkingPoint::query()
                ->where('one_on_one_entry_id', $entry->id)
                ->where('id', $childId)
                ->firstOrFail()
                ->delete(),
            'action_item' => OneOnOneActionItem::query()
                ->where('one_on_one_entry_id', $entry->id)
                ->where('id', $childId)
                ->firstOrFail()
                ->delete(),
            default => OneOnOneNote::query()
                ->where('one_on_one_entry_id', $entry->id)
                ->where('id', $childId)
                ->firstOrFail()
                ->delete(),
        };

        return ApiResponse::success(null);
    }

    // --- Rate your manager ---

    public function pendingRateAnswers(Request $request): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        return ApiResponse::success($this->grow->pendingRateAnswers($actor));
    }

    public function submitRating(Request $request, string $companyId, string $answerId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $answer = RateYourManagerAnswer::query()
            ->where('id', $answerId)
            ->whereHas('survey', fn ($q) => $q->where('company_id', $company->id))
            ->firstOrFail();

        $data = $request->validate([
            'rating' => ['required', 'string', 'in:bad,average,good'],
        ]);

        try {
            $answer = $this->grow->submitRating($actor, $answer, $data['rating']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->grow->rateAnswerPayload($answer));
    }

    public function commentOnRating(Request $request, string $companyId, string $answerId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $answer = RateYourManagerAnswer::query()
            ->where('id', $answerId)
            ->whereHas('survey', fn ($q) => $q->where('company_id', $company->id))
            ->firstOrFail();

        $data = $request->validate([
            'comment' => ['nullable', 'string', 'max:65535'],
            'reveal_identity_to_manager' => ['sometimes', 'boolean'],
        ]);

        try {
            $answer = $this->grow->commentOnRating(
                $actor,
                $answer,
                $data['comment'] ?? null,
                (bool) ($data['reveal_identity_to_manager'] ?? false),
            );
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->grow->rateAnswerPayload($answer));
    }

    public function managerSurveys(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $manager = Employee::query()->where('company_id', $company->id)->where('id', $employeeId)->firstOrFail();

        try {
            return ApiResponse::success($this->grow->surveysForManager($company, $actor, $manager));
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }
    }

    // --- Skills ---

    public function listSkills(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return ApiResponse::success($this->grow->listCompanySkills($company));
    }

    public function showSkill(Request $request, string $companyId, string $skillId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $skill = Skill::query()->where('company_id', $company->id)->where('id', $skillId)->firstOrFail();

        return ApiResponse::success($this->grow->skillDetail($skill));
    }

    public function updateSkill(Request $request, string $companyId, string $skillId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $skill = Skill::query()->where('company_id', $company->id)->where('id', $skillId)->firstOrFail();
        $data = $request->validate(['name' => ['required', 'string', 'max:255']]);

        try {
            $skill = $this->grow->updateSkill($company, $skill, $data['name']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success([
            'id' => (string) $skill->id,
            'name' => $skill->name,
        ]);
    }

    public function destroySkill(Request $request, string $companyId, string $skillId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $skill = Skill::query()->where('company_id', $company->id)->where('id', $skillId)->firstOrFail();
        $this->grow->destroySkill($skill);

        return ApiResponse::success(null);
    }

    public function employeeSkills(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $subject = Employee::query()->where('company_id', $company->id)->where('id', $employeeId)->firstOrFail();

        return ApiResponse::success($this->grow->employeeSkills($subject));
    }

    public function attachEmployeeSkill(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $subject = Employee::query()->where('company_id', $company->id)->where('id', $employeeId)->firstOrFail();
        $data = $request->validate(['name' => ['required', 'string', 'max:255']]);

        try {
            $skill = $this->grow->attachSkill($company, $actor, $subject, $data['name']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success([
            'id' => (string) $skill->id,
            'name' => $skill->name,
        ]);
    }

    public function detachEmployeeSkill(Request $request, string $companyId, string $employeeId, string $skillId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $subject = Employee::query()->where('company_id', $company->id)->where('id', $employeeId)->firstOrFail();
        $skill = Skill::query()->where('company_id', $company->id)->where('id', $skillId)->firstOrFail();

        try {
            $this->grow->detachSkill($actor, $subject, $skill);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null);
    }

    public function searchSkills(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $q = (string) $request->query('q', '');

        return ApiResponse::success($this->grow->searchSkills($company, $q));
    }

    // --- e-Coffee ---

    public function getECoffee(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return ApiResponse::success(['enabled' => $this->grow->getECoffeeEnabled($company)]);
    }

    public function updateECoffee(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        $data = $request->validate(['enabled' => ['required', 'boolean']]);
        $company = $this->grow->setECoffeeEnabled($company, (bool) $data['enabled']);

        return ApiResponse::success(['enabled' => (bool) $company->e_coffee_enabled]);
    }

    public function currentECoffee(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        return ApiResponse::success($this->grow->currentECoffeeMatch($company, $actor));
    }

    public function markECoffeeHappened(Request $request, string $companyId, string $matchId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $match = ECoffeeMatch::query()
            ->where('id', $matchId)
            ->whereHas('eCoffee', fn ($q) => $q->where('company_id', $company->id))
            ->firstOrFail();

        try {
            $match = $this->grow->markECoffeeHappened($actor, $match);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->grow->eCoffeeMatchPayload($match));
    }

    public function employeeECoffeeHistory(Request $request, string $companyId, string $employeeId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $subject = Employee::query()->where('company_id', $company->id)->where('id', $employeeId)->firstOrFail();

        try {
            return ApiResponse::success($this->grow->employeeECoffeeHistory($company, $actor, $subject));
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }
    }

    // --- Discipline ---

    public function listDisciplineCases(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $active = $request->query('active');
        $activeBool = $active === null ? null : filter_var($active, FILTER_VALIDATE_BOOLEAN, FILTER_NULL_ON_FAILURE);

        return ApiResponse::success($this->grow->listDisciplineCases($company, $actor, $activeBool));
    }

    public function showDisciplineCase(Request $request, string $companyId, string $caseId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $case = DisciplineCase::query()
            ->where('company_id', $company->id)
            ->where('id', $caseId)
            ->firstOrFail();

        try {
            $this->grow->ensureDisciplineAccess($actor, $case);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->grow->disciplineCasePayload($case, true));
    }

    public function storeDisciplineCase(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $data = $request->validate(['employee_id' => ['required', 'uuid', 'exists:employees,id']]);
        $subject = Employee::query()->where('company_id', $company->id)->where('id', $data['employee_id'])->firstOrFail();

        try {
            $case = $this->grow->createDisciplineCase($company, $actor, $subject);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->grow->disciplineCasePayload($case, true));
    }

    public function toggleDisciplineCase(Request $request, string $companyId, string $caseId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $case = DisciplineCase::query()->where('company_id', $company->id)->where('id', $caseId)->firstOrFail();

        try {
            $case = $this->grow->toggleDisciplineCase($actor, $case);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($this->grow->disciplineCasePayload($case, true));
    }

    public function destroyDisciplineCase(Request $request, string $companyId, string $caseId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $case = DisciplineCase::query()->where('company_id', $company->id)->where('id', $caseId)->firstOrFail();

        try {
            $this->grow->destroyDisciplineCase($actor, $case);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null);
    }

    public function storeDisciplineEvent(Request $request, string $companyId, string $caseId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $case = DisciplineCase::query()->where('company_id', $company->id)->where('id', $caseId)->firstOrFail();
        $data = $request->validate([
            'happened_at' => ['required', 'date'],
            'description' => ['required', 'string'],
        ]);

        try {
            $event = $this->grow->createDisciplineEvent($actor, $case, $data['happened_at'], $data['description']);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success([
            'id' => (string) $event->id,
            'author_name' => $event->author_name,
            'happened_at' => $event->happened_at?->toDateString(),
            'description' => $event->description,
            'files' => [],
            'created_at' => $event->created_at?->toIso8601String(),
        ]);
    }

    public function destroyDisciplineEvent(Request $request, string $companyId, string $caseId, string $eventId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $case = DisciplineCase::query()->where('company_id', $company->id)->where('id', $caseId)->firstOrFail();
        $event = DisciplineEvent::query()
            ->where('discipline_case_id', $case->id)
            ->where('id', $eventId)
            ->firstOrFail();

        try {
            $this->grow->destroyDisciplineEvent($actor, $event);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null);
    }

    public function attachDisciplineFile(Request $request, string $companyId, string $caseId, string $eventId): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $case = DisciplineCase::query()->where('company_id', $company->id)->where('id', $caseId)->firstOrFail();
        $event = DisciplineEvent::query()
            ->where('discipline_case_id', $case->id)
            ->where('id', $eventId)
            ->firstOrFail();

        $data = $request->validate([
            'temporary_upload_id' => ['required', 'integer', 'exists:temporary_uploads,id'],
            'media_id' => ['required', 'integer'],
        ]);

        try {
            $media = $this->grow->attachDisciplineFile(
                $actor,
                $event,
                (int) $data['temporary_upload_id'],
                (int) $data['media_id'],
            );
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success([
            'id' => $media->id,
            'file_name' => $media->file_name,
            'mime_type' => $media->mime_type,
            'size' => $media->size,
            'url' => url('/api/v1/media/'.$media->id.'/file'),
        ]);
    }
}
