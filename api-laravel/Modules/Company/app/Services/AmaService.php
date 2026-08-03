<?php

namespace Modules\Company\Services;

use Illuminate\Support\Collection;
use Illuminate\Support\Facades\DB;
use Modules\Company\Models\AskMeAnythingQuestion;
use Modules\Company\Models\AskMeAnythingSession;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use RuntimeException;

final class AmaService
{
    public function list(Company $company): Collection
    {
        return AskMeAnythingSession::query()->with('questions')->where('company_id', $company->id)->orderByDesc('happened_at')->get();
    }

    public function create(Company $company, array $data): AskMeAnythingSession
    {
        $session = AskMeAnythingSession::query()->create(['company_id' => $company->id, ...$data, 'active' => false]);
        if ($data['active'] ?? false) {
            return $this->activate($session);
        }

        return $session;
    }

    public function update(AskMeAnythingSession $session, array $data): AskMeAnythingSession
    {
        $activate = $data['active'] ?? null;
        unset($data['active']);
        $session->fill($data)->save();
        if ($activate === true) {
            return $this->activate($session);
        }
        if ($activate === false) {
            $session->update(['active' => false]);
        }

        return $session->fresh('questions');
    }

    public function delete(AskMeAnythingSession $session): void { $session->delete(); }

    public function activate(AskMeAnythingSession $session): AskMeAnythingSession
    {
        return DB::transaction(function () use ($session) {
            AskMeAnythingSession::query()->where('company_id', $session->company_id)->where('id', '!=', $session->id)->update(['active' => false]);
            $session->update(['active' => true]);

            return $session->fresh('questions');
        });
    }

    public function ask(AskMeAnythingSession $session, Employee $actor, array $data): AskMeAnythingQuestion
    {
        if (! $session->active) {
            throw new RuntimeException('AMA session is not active', 409);
        }

        return AskMeAnythingQuestion::query()->create([
            'ask_me_anything_session_id' => $session->id,
            'employee_id' => ($data['anonymous'] ?? false) ? null : $actor->id,
            'question' => $data['question'],
            'anonymous' => $data['anonymous'] ?? false,
            'answered' => false,
        ]);
    }

    public function markAnswered(AskMeAnythingQuestion $question): AskMeAnythingQuestion
    {
        $question->update(['answered' => true]);

        return $question->fresh();
    }
}
