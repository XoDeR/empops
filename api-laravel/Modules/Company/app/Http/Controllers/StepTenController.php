<?php

namespace Modules\Company\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\AskMeAnythingQuestion;
use Modules\Company\Models\AskMeAnythingSession;
use Modules\Company\Models\Company;
use Modules\Company\Models\Flow;
use Modules\Company\Models\FlowAction;
use Modules\Company\Models\FlowStep;
use Modules\Company\Models\Wiki;
use Modules\Company\Models\WikiPage;
use Modules\Company\Services\AmaService;
use Modules\Company\Services\FlowService;
use Modules\Company\Services\WikiService;
use Modules\Employee\Models\Employee;
use RuntimeException;

class StepTenController extends Controller
{
    public function __construct(
        private readonly FlowService $flows,
        private readonly WikiService $wikis,
        private readonly AmaService $ama,
    ) {}

    public function flows(Request $r): JsonResponse { return ApiResponse::success($this->flows->list($this->company($r))); }
    public function flow(Request $r, string $companyId, string $flowId): JsonResponse { return ApiResponse::success($this->flowModel($r, $flowId)->load('steps.actions')); }

    public function createFlow(Request $r): JsonResponse
    {
        $data = $r->validate(['name' => ['required', 'string', 'max:255'], 'type' => ['required', 'in:employee_joins_company,employee_leaves_company']]);

        return ApiResponse::success($this->flows->create($this->company($r), $data), 'Flow created', 201);
    }

    public function updateFlow(Request $r, string $companyId, string $flowId): JsonResponse
    {
        $data = $r->validate(['name' => ['sometimes', 'required', 'string', 'max:255'], 'type' => ['sometimes', 'required', 'in:employee_joins_company,employee_leaves_company']]);

        return ApiResponse::success($this->flows->update($this->flowModel($r, $flowId), $data), 'Flow updated');
    }

    public function deleteFlow(Request $r, string $companyId, string $flowId): JsonResponse
    {
        $this->flows->delete($this->flowModel($r, $flowId));

        return ApiResponse::success(null, 'Flow deleted');
    }

    public function addStep(Request $r, string $companyId, string $flowId): JsonResponse
    {
        $data = $r->validate(['number' => ['required', 'integer', 'min:0'], 'unit_of_time' => ['required', 'in:days,weeks,months'], 'modifier' => ['required', 'in:before,after,same_day']]);

        return ApiResponse::success($this->flows->addStep($this->flowModel($r, $flowId), $data), 'Flow step created', 201);
    }

    public function deleteStep(Request $r, string $companyId, string $flowId, string $stepId): JsonResponse
    {
        $this->flows->removeStep($this->step($r, $flowId, $stepId));

        return ApiResponse::success(null, 'Flow step deleted');
    }

    public function addAction(Request $r, string $companyId, string $flowId, string $stepId): JsonResponse
    {
        $data = $r->validate(['type' => ['required', 'in:notification'], 'recipient' => ['required', 'in:employee,manager,hr'], 'specific_recipient_information' => ['nullable', 'string']]);

        return ApiResponse::success($this->flows->addAction($this->step($r, $flowId, $stepId), $data), 'Flow action created', 201);
    }

    public function deleteAction(Request $r, string $companyId, string $flowId, string $stepId, string $actionId): JsonResponse
    {
        $step = $this->step($r, $flowId, $stepId);
        $action = FlowAction::query()->where('step_id', $step->id)->findOrFail($actionId);
        $this->flows->removeAction($action);

        return ApiResponse::success(null, 'Flow action deleted');
    }

    public function wikis(Request $r): JsonResponse { return ApiResponse::success($this->wikis->list($this->company($r))); }
    public function wiki(Request $r, string $companyId, string $wikiId): JsonResponse { return ApiResponse::success($this->wikiModel($r, $wikiId)->load('pages')); }

    public function createWiki(Request $r): JsonResponse
    {
        return ApiResponse::success($this->wikis->create($this->company($r), $r->validate(['title' => ['required', 'string', 'max:255']])), 'Wiki created', 201);
    }

    public function updateWiki(Request $r, string $companyId, string $wikiId): JsonResponse
    {
        return ApiResponse::success($this->wikis->update($this->wikiModel($r, $wikiId), $r->validate(['title' => ['required', 'string', 'max:255']])), 'Wiki updated');
    }

    public function deleteWiki(Request $r, string $companyId, string $wikiId): JsonResponse
    {
        $this->wikis->delete($this->wikiModel($r, $wikiId));

        return ApiResponse::success(null, 'Wiki deleted');
    }

    public function createPage(Request $r, string $companyId, string $wikiId): JsonResponse
    {
        $data = $r->validate(['title' => ['required', 'string', 'max:255'], 'content' => ['nullable', 'string']]);

        return ApiResponse::success($this->wikis->createPage($this->wikiModel($r, $wikiId), $data, $this->actor($r)), 'Wiki page created', 201);
    }

    public function page(Request $r, string $companyId, string $wikiId, string $pageId): JsonResponse
    {
        return ApiResponse::success($this->wikis->showPage($this->pageModel($r, $wikiId, $pageId)));
    }

    public function updatePage(Request $r, string $companyId, string $wikiId, string $pageId): JsonResponse
    {
        $data = $r->validate(['title' => ['sometimes', 'required', 'string', 'max:255'], 'content' => ['nullable', 'string']]);

        return ApiResponse::success($this->wikis->updatePage($this->pageModel($r, $wikiId, $pageId), $data, $this->actor($r)), 'Wiki page updated');
    }

    public function deletePage(Request $r, string $companyId, string $wikiId, string $pageId): JsonResponse
    {
        $this->wikis->deletePage($this->pageModel($r, $wikiId, $pageId));

        return ApiResponse::success(null, 'Wiki page deleted');
    }

    public function amaSessions(Request $r): JsonResponse { return ApiResponse::success($this->ama->list($this->company($r))); }

    public function amaSession(Request $r, string $companyId, string $sessionId): JsonResponse
    {
        return ApiResponse::success($this->session($r, $sessionId)->load('questions'));
    }

    public function pageRevisions(Request $r, string $companyId, string $wikiId, string $pageId): JsonResponse
    {
        return ApiResponse::success($this->pageModel($r, $wikiId, $pageId)->revisions()->get());
    }

    public function createAma(Request $r): JsonResponse
    {
        $data = $r->validate(['happened_at' => ['required', 'date'], 'theme' => ['nullable', 'string', 'max:255'], 'active' => ['sometimes', 'boolean']]);

        return ApiResponse::success($this->ama->create($this->company($r), $data), 'AMA session created', 201);
    }

    public function updateAma(Request $r, string $companyId, string $sessionId): JsonResponse
    {
        $data = $r->validate(['happened_at' => ['sometimes', 'date'], 'theme' => ['nullable', 'string', 'max:255'], 'active' => ['sometimes', 'boolean']]);

        return ApiResponse::success($this->ama->update($this->session($r, $sessionId), $data), 'AMA session updated');
    }

    public function deleteAma(Request $r, string $companyId, string $sessionId): JsonResponse
    {
        $this->ama->delete($this->session($r, $sessionId));

        return ApiResponse::success(null, 'AMA session deleted');
    }

    public function ask(Request $r, string $companyId, string $sessionId): JsonResponse
    {
        $data = $r->validate(['question' => ['required', 'string'], 'anonymous' => ['sometimes', 'boolean']]);
        try {
            $question = $this->ama->ask($this->session($r, $sessionId), $this->actor($r), $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($question, 'Question submitted', 201);
    }

    public function answer(Request $r, string $companyId, string $sessionId, string $questionId): JsonResponse
    {
        $session = $this->session($r, $sessionId);
        $question = AskMeAnythingQuestion::query()->where('ask_me_anything_session_id', $session->id)->findOrFail($questionId);
        $data = $r->validate(['answered' => ['sometimes', 'boolean']]);
        if (($data['answered'] ?? true) === false) {
            $question->update(['answered' => false]);

            return ApiResponse::success($question->fresh(), 'Question updated');
        }

        return ApiResponse::success($this->ama->markAnswered($question), 'Question marked answered');
    }

    private function company(Request $r): Company { return $r->attributes->get('company'); }
    private function actor(Request $r): Employee { return $r->attributes->get('employee'); }
    private function flowModel(Request $r, string $id): Flow { return Flow::query()->where('company_id', $this->company($r)->id)->findOrFail($id); }
    private function wikiModel(Request $r, string $id): Wiki { return Wiki::query()->where('company_id', $this->company($r)->id)->findOrFail($id); }
    private function session(Request $r, string $id): AskMeAnythingSession { return AskMeAnythingSession::query()->where('company_id', $this->company($r)->id)->findOrFail($id); }

    private function step(Request $r, string $flowId, string $stepId): FlowStep
    {
        return FlowStep::query()->where('flow_id', $this->flowModel($r, $flowId)->id)->findOrFail($stepId);
    }

    private function pageModel(Request $r, string $wikiId, string $pageId): WikiPage
    {
        return WikiPage::query()->where('wiki_id', $this->wikiModel($r, $wikiId)->id)->findOrFail($pageId);
    }
}
