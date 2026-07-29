<?php

namespace Modules\Company\Services;

use Illuminate\Support\Facades\DB;
use Modules\Company\Models\Answer;
use Modules\Company\Models\Company;
use Modules\Company\Models\Question;
use Modules\Employee\Models\Employee;

final class QuestionService
{
    public function __construct(private readonly AuditLogger $audit) {}

    /**
     * @param  array{title: string}  $data
     */
    public function create(Company $company, array $data, Employee $actor): Question
    {
        $question = Question::query()->create([
            'company_id' => $company->id,
            'title' => trim($data['title']),
            'active' => false,
        ]);

        $this->audit->log($company, $actor, 'question.created', $question);

        return $question;
    }

    /**
     * @param  array{title?: string}  $data
     */
    public function update(Question $question, array $data, Employee $actor): Question
    {
        if (isset($data['title'])) {
            $question->title = trim($data['title']);
            $question->save();
        }

        $this->audit->log($question->company, $actor, 'question.updated', $question);

        return $question->fresh();
    }

    public function destroy(Question $question, Employee $actor): void
    {
        $company = $question->company;
        $payload = ['question_id' => (string) $question->id, 'title' => $question->title];
        $question->delete();
        $this->audit->log($company, $actor, 'question.deleted', null, $payload);
    }

    public function activate(Question $question, Employee $actor): Question
    {
        return DB::transaction(function () use ($question, $actor) {
            Question::query()
                ->where('company_id', $question->company_id)
                ->where('active', true)
                ->where('id', '!=', $question->id)
                ->update([
                    'active' => false,
                    'deactivated_at' => now(),
                ]);

            $question->active = true;
            $question->activated_at = now();
            $question->deactivated_at = null;
            $question->save();

            $this->audit->log($question->company, $actor, 'question.activated', $question);

            return $question->fresh();
        });
    }

    public function deactivate(Question $question, Employee $actor): Question
    {
        $question->active = false;
        $question->deactivated_at = now();
        $question->save();

        $this->audit->log($question->company, $actor, 'question.deactivated', $question);

        return $question->fresh();
    }

    /**
     * @param  array{body: string}  $data
     */
    public function upsertAnswer(Question $question, Employee $actor, array $data): Answer
    {
        $answer = Answer::query()->updateOrCreate(
            [
                'question_id' => $question->id,
                'employee_id' => $actor->id,
            ],
            [
                'company_id' => $question->company_id,
                'body' => trim($data['body']),
            ],
        );

        return $answer->fresh(['employee']);
    }

    public function updateAnswer(Answer $answer, array $data): Answer
    {
        if (isset($data['body'])) {
            $answer->body = trim($data['body']);
            $answer->save();
        }

        return $answer->fresh(['employee']);
    }

    public function destroyAnswer(Answer $answer): void
    {
        $answer->delete();
    }

    /**
     * @return array<string, mixed>
     */
    public function listPayload(Question $question, ?Employee $actor = null): array
    {
        return [
            'id' => (string) $question->id,
            'company_id' => (string) $question->company_id,
            'title' => $question->title,
            'active' => (bool) $question->active,
            'activated_at' => $question->activated_at?->toIso8601String(),
            'deactivated_at' => $question->deactivated_at?->toIso8601String(),
            'answer_count' => $question->answers()->count(),
            'created_at' => $question->created_at?->toIso8601String(),
            'updated_at' => $question->updated_at?->toIso8601String(),
        ];
    }

    /**
     * @return array<string, mixed>
     */
    public function detailPayload(Question $question, ?Employee $actor = null): array
    {
        $question->loadMissing(['answers.employee']);

        $myAnswer = null;
        if ($actor !== null) {
            $mine = $question->answers->firstWhere('employee_id', $actor->id);
            $myAnswer = $mine ? $this->answerPayload($mine) : null;
        }

        return array_merge($this->listPayload($question, $actor), [
            'answers' => $question->answers
                ->map(fn (Answer $a) => $this->answerPayload($a))
                ->values()
                ->all(),
            'my_answer' => $myAnswer,
        ]);
    }

    /**
     * @return array<string, mixed>|null
     */
    public function activePayload(Company $company, Employee $actor): ?array
    {
        $question = Question::query()
            ->where('company_id', $company->id)
            ->where('active', true)
            ->first();

        if ($question === null) {
            return null;
        }

        $mine = Answer::query()
            ->where('question_id', $question->id)
            ->where('employee_id', $actor->id)
            ->first();

        return [
            'id' => (string) $question->id,
            'company_id' => (string) $question->company_id,
            'title' => $question->title,
            'active' => true,
            'answered' => $mine !== null,
            'my_answer' => $mine ? $this->answerPayload($mine) : null,
            'activated_at' => $question->activated_at?->toIso8601String(),
        ];
    }

    /**
     * @return array<string, mixed>
     */
    public function answerPayload(Answer $answer): array
    {
        $answer->loadMissing('employee');

        return [
            'id' => (string) $answer->id,
            'question_id' => (string) $answer->question_id,
            'employee_id' => (string) $answer->employee_id,
            'body' => $answer->body,
            'employee' => $answer->employee ? [
                'id' => (string) $answer->employee->id,
                'first_name' => $answer->employee->first_name,
                'last_name' => $answer->employee->last_name,
                'email' => $answer->employee->email,
            ] : null,
            'created_at' => $answer->created_at?->toIso8601String(),
            'updated_at' => $answer->updated_at?->toIso8601String(),
        ];
    }
}
