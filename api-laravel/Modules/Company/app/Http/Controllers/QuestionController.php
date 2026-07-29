<?php

namespace Modules\Company\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Answer;
use Modules\Company\Models\Company;
use Modules\Company\Models\Question;
use Modules\Company\Services\QuestionService;
use Modules\Employee\Models\Employee;

class QuestionController extends Controller
{
    public function __construct(private readonly QuestionService $questions) {}

    public function index(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $list = Question::query()
            ->where('company_id', $company->id)
            ->orderByDesc('created_at')
            ->get()
            ->map(fn (Question $q) => $this->questions->listPayload($q, $actor))
            ->values()
            ->all();

        return ApiResponse::success($list);
    }

    public function active(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        return ApiResponse::success($this->questions->activePayload($company, $actor));
    }

    public function show(Request $request, string $companyId, string $questionId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $question = $this->findInCompany($request, $questionId);

        return ApiResponse::success($this->questions->detailPayload($question, $actor));
    }

    public function store(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $validated = $request->validate([
            'title' => ['required', 'string'],
        ]);

        $question = $this->questions->create($company, $validated, $actor);

        return ApiResponse::success($this->questions->listPayload($question, $actor), 'Question created', 201);
    }

    public function update(Request $request, string $companyId, string $questionId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $question = $this->findInCompany($request, $questionId);

        $validated = $request->validate([
            'title' => ['sometimes', 'string'],
        ]);

        $updated = $this->questions->update($question, $validated, $actor);

        return ApiResponse::success($this->questions->listPayload($updated, $actor), 'Question updated');
    }

    public function destroy(Request $request, string $companyId, string $questionId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $question = $this->findInCompany($request, $questionId);
        $this->questions->destroy($question, $actor);

        return ApiResponse::success(null, 'Question deleted');
    }

    public function activate(Request $request, string $companyId, string $questionId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $question = $this->findInCompany($request, $questionId);
        $updated = $this->questions->activate($question, $actor);

        return ApiResponse::success($this->questions->listPayload($updated, $actor), 'Question activated');
    }

    public function deactivate(Request $request, string $companyId, string $questionId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $question = $this->findInCompany($request, $questionId);
        $updated = $this->questions->deactivate($question, $actor);

        return ApiResponse::success($this->questions->listPayload($updated, $actor), 'Question deactivated');
    }

    public function storeAnswer(Request $request, string $companyId, string $questionId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $question = $this->findInCompany($request, $questionId);

        $validated = $request->validate([
            'body' => ['required', 'string'],
        ]);

        $answer = $this->questions->upsertAnswer($question, $actor, $validated);

        return ApiResponse::success($this->questions->answerPayload($answer), 'Answer saved', 201);
    }

    public function updateAnswer(Request $request, string $companyId, string $questionId, string $answerId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $answer = $this->findAnswer($request, $questionId, $answerId);

        if ((string) $answer->employee_id !== (string) $actor->id && ! $actor->hasPermissionTo('questions.update')) {
            return ApiResponse::error('Forbidden', 403);
        }

        $validated = $request->validate([
            'body' => ['required', 'string'],
        ]);

        $updated = $this->questions->updateAnswer($answer, $validated);

        return ApiResponse::success($this->questions->answerPayload($updated), 'Answer updated');
    }

    public function destroyAnswer(Request $request, string $companyId, string $questionId, string $answerId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $answer = $this->findAnswer($request, $questionId, $answerId);

        if ((string) $answer->employee_id !== (string) $actor->id && ! $actor->hasPermissionTo('questions.update')) {
            return ApiResponse::error('Forbidden', 403);
        }

        $this->questions->destroyAnswer($answer);

        return ApiResponse::success(null, 'Answer deleted');
    }

    private function findInCompany(Request $request, string $questionId): Question
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return Question::query()
            ->where('company_id', $company->id)
            ->where('id', $questionId)
            ->firstOrFail();
    }

    private function findAnswer(Request $request, string $questionId, string $answerId): Answer
    {
        $question = $this->findInCompany($request, $questionId);

        return Answer::query()
            ->where('question_id', $question->id)
            ->where('id', $answerId)
            ->firstOrFail();
    }
}
