<?php

namespace Modules\Grow\Services;

use Carbon\Carbon;
use Illuminate\Support\Collection;
use Illuminate\Support\Str;
use Modules\Company\Models\Company;
use Modules\Employee\Models\DirectReport;
use Modules\Employee\Models\Employee;
use Modules\Grow\Models\DisciplineCase;
use Modules\Grow\Models\DisciplineEvent;
use Modules\Grow\Models\ECoffee;
use Modules\Grow\Models\ECoffeeMatch;
use Modules\Grow\Models\Morale;
use Modules\Grow\Models\MoraleCompanyHistory;
use Modules\Grow\Models\MoraleTeamHistory;
use Modules\Grow\Models\OneOnOneActionItem;
use Modules\Grow\Models\OneOnOneEntry;
use Modules\Grow\Models\OneOnOneNote;
use Modules\Grow\Models\OneOnOneTalkingPoint;
use Modules\Grow\Models\RateYourManagerAnswer;
use Modules\Grow\Models\RateYourManagerSurvey;
use Modules\Grow\Models\Skill;
use Modules\Notification\Services\NotificationService;
use Modules\Team\Models\Team;
use Modules\Uploads\Services\MediaAttachService;
use RuntimeException;
use Spatie\MediaLibrary\MediaCollections\Models\Media;

final class GrowService
{
    public function __construct(
        private readonly MediaAttachService $mediaAttach,
        private readonly NotificationService $notifications,
    ) {}

    // --- Auth helpers ---

    public function isHr(Employee $actor): bool
    {
        return $actor->hasAnyRole(['administrator', 'hr']);
    }

    public function isManagerOf(Employee $actor, Employee $subject): bool
    {
        return DirectReport::query()
            ->where('company_id', $actor->company_id)
            ->where('manager_id', $actor->id)
            ->where('employee_id', $subject->id)
            ->exists();
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

    // --- Morale ---

    public function todayMorale(Employee $employee): ?Morale
    {
        return Morale::query()
            ->where('employee_id', $employee->id)
            ->whereDate('created_at', Carbon::today())
            ->first();
    }

    public function logMorale(Company $company, Employee $actor, int $emotion, ?string $comment = null): Morale
    {
        if (! in_array($emotion, [1, 2, 3], true)) {
            throw new RuntimeException('Emotion must be 1, 2, or 3', 422);
        }

        if ($this->todayMorale($actor) !== null) {
            throw new RuntimeException('Morale already logged today', 409);
        }

        return Morale::query()->create([
            'company_id' => $company->id,
            'employee_id' => $actor->id,
            'emotion' => $emotion,
            'comment' => $comment,
        ]);
    }

    public function moralePayload(Morale $morale): array
    {
        return [
            'id' => (string) $morale->id,
            'employee_id' => (string) $morale->employee_id,
            'emotion' => (int) $morale->emotion,
            'comment' => $morale->comment,
            'created_at' => $morale->created_at?->toIso8601String(),
        ];
    }

    public function companyMoraleHistory(Company $company, int $limit = 30): array
    {
        return MoraleCompanyHistory::query()
            ->where('company_id', $company->id)
            ->orderByDesc('created_at')
            ->limit($limit)
            ->get()
            ->map(fn (MoraleCompanyHistory $h) => [
                'id' => (string) $h->id,
                'average' => (float) $h->average,
                'number_of_employees' => (int) $h->number_of_employees,
                'created_at' => $h->created_at?->toIso8601String(),
            ])
            ->values()
            ->all();
    }

    public function teamMoraleHistory(Team $team, int $limit = 30): array
    {
        return MoraleTeamHistory::query()
            ->where('team_id', $team->id)
            ->orderByDesc('created_at')
            ->limit($limit)
            ->get()
            ->map(fn (MoraleTeamHistory $h) => [
                'id' => (string) $h->id,
                'average' => (float) $h->average,
                'number_of_team_members' => (int) $h->number_of_team_members,
                'created_at' => $h->created_at?->toIso8601String(),
            ])
            ->values()
            ->all();
    }

    public function employeeMorale(Company $company, Employee $actor, Employee $subject, int $limit = 30): array
    {
        if ($actor->id !== $subject->id && ! $this->isHr($actor) && ! $this->isManagerOf($actor, $subject)) {
            throw new RuntimeException('Forbidden', 403);
        }

        return Morale::query()
            ->where('company_id', $company->id)
            ->where('employee_id', $subject->id)
            ->orderByDesc('created_at')
            ->limit($limit)
            ->get()
            ->map(fn (Morale $m) => $this->moralePayload($m))
            ->values()
            ->all();
    }

    public function logCompanyMoraleForDate(?Carbon $date = null): int
    {
        $date = ($date ?? Carbon::today())->startOfDay();
        $created = 0;

        Company::query()->chunkById(50, function ($companies) use ($date, &$created) {
            foreach ($companies as $company) {
                $exists = MoraleCompanyHistory::query()
                    ->where('company_id', $company->id)
                    ->whereDate('created_at', $date)
                    ->exists();
                if ($exists) {
                    continue;
                }

                $avg = Morale::query()
                    ->where('company_id', $company->id)
                    ->whereDate('created_at', $date)
                    ->avg('emotion');
                $count = Morale::query()
                    ->where('company_id', $company->id)
                    ->whereDate('created_at', $date)
                    ->count();

                $row = MoraleCompanyHistory::query()->create([
                    'company_id' => $company->id,
                    'average' => $avg !== null ? (float) $avg : 0.0,
                    'number_of_employees' => $count,
                ]);
                $row->created_at = $date->copy();
                $row->updated_at = $date->copy();
                $row->save();
                $created++;
            }
        });

        return $created;
    }

    public function logTeamMoraleForDate(?Carbon $date = null): int
    {
        $date = ($date ?? Carbon::today())->startOfDay();
        $created = 0;

        Team::query()->chunkById(50, function ($teams) use ($date, &$created) {
            foreach ($teams as $team) {
                $exists = MoraleTeamHistory::query()
                    ->where('team_id', $team->id)
                    ->whereDate('created_at', $date)
                    ->exists();
                if ($exists) {
                    continue;
                }

                $memberIds = $team->employees()->pluck('employees.id');
                $query = Morale::query()
                    ->whereIn('employee_id', $memberIds)
                    ->whereDate('created_at', $date);
                $avg = (clone $query)->avg('emotion');
                $count = (clone $query)->count();

                $row = MoraleTeamHistory::query()->create([
                    'team_id' => $team->id,
                    'average' => $avg !== null ? (float) $avg : 0.0,
                    'number_of_team_members' => $count,
                ]);
                $row->created_at = $date->copy();
                $row->updated_at = $date->copy();
                $row->save();
                $created++;
            }
        });

        return $created;
    }

    // --- One-on-ones ---

    public function ensureOneOnOneParticipant(Employee $actor, OneOnOneEntry $entry): void
    {
        if ($actor->id === $entry->manager_id || $actor->id === $entry->employee_id || $this->isHr($actor)) {
            return;
        }
        throw new RuntimeException('Forbidden', 403);
    }

    public function createOrGetOpenOneOnOne(Company $company, Employee $manager, Employee $employee): OneOnOneEntry
    {
        if ($manager->id === $employee->id) {
            throw new RuntimeException('Manager and employee must differ', 422);
        }

        if (! $this->isManagerOf($manager, $employee) && ! $this->isHr($manager)) {
            // When called by HR creating for a pair, manager must still manage employee
            if (! DirectReport::query()
                ->where('company_id', $company->id)
                ->where('manager_id', $manager->id)
                ->where('employee_id', $employee->id)
                ->exists()) {
                throw new RuntimeException('Not a manager of this employee', 422);
            }
        }

        $open = OneOnOneEntry::query()
            ->where('company_id', $company->id)
            ->where('manager_id', $manager->id)
            ->where('employee_id', $employee->id)
            ->where('happened', false)
            ->first();

        if ($open !== null) {
            return $open->load(['talkingPoints', 'actionItems', 'notes', 'manager', 'employee']);
        }

        return $this->createOneOnOneEntry($company, $manager, $employee);
    }

    public function createOneOnOneEntry(Company $company, Employee $manager, Employee $employee): OneOnOneEntry
    {
        $entry = OneOnOneEntry::query()->create([
            'company_id' => $company->id,
            'manager_id' => $manager->id,
            'employee_id' => $employee->id,
            'happened' => false,
        ]);

        $previous = OneOnOneEntry::query()
            ->where('company_id', $company->id)
            ->where('manager_id', $manager->id)
            ->where('employee_id', $employee->id)
            ->where('id', '!=', $entry->id)
            ->orderByDesc('created_at')
            ->first();

        if ($previous !== null) {
            $unchecked = OneOnOneActionItem::query()
                ->where('one_on_one_entry_id', $previous->id)
                ->where('checked', false)
                ->get();
            foreach ($unchecked as $item) {
                OneOnOneTalkingPoint::query()->create([
                    'one_on_one_entry_id' => $entry->id,
                    'description' => $item->description,
                    'checked' => false,
                ]);
            }
        }

        return $entry->load(['talkingPoints', 'actionItems', 'notes', 'manager', 'employee']);
    }

    public function markOneOnOneHappened(OneOnOneEntry $entry): OneOnOneEntry
    {
        if ($entry->happened) {
            throw new RuntimeException('One-on-one already marked happened', 409);
        }

        $entry->happened = true;
        $entry->happened_at = now();
        $entry->save();

        $manager = Employee::query()->findOrFail($entry->manager_id);
        $employee = Employee::query()->findOrFail($entry->employee_id);
        $company = Company::query()->findOrFail($entry->company_id);

        $this->createOneOnOneEntry($company, $manager, $employee);

        return $entry->fresh(['talkingPoints', 'actionItems', 'notes', 'manager', 'employee']);
    }

    public function oneOnOnePayload(OneOnOneEntry $entry): array
    {
        $entry->loadMissing(['talkingPoints', 'actionItems', 'notes', 'manager', 'employee']);

        return [
            'id' => (string) $entry->id,
            'company_id' => (string) $entry->company_id,
            'manager' => $this->employeeSummary($entry->manager),
            'employee' => $this->employeeSummary($entry->employee),
            'happened' => (bool) $entry->happened,
            'happened_at' => $entry->happened_at?->toIso8601String(),
            'talking_points' => $entry->talkingPoints->map(fn (OneOnOneTalkingPoint $p) => [
                'id' => (string) $p->id,
                'description' => $p->description,
                'checked' => (bool) $p->checked,
            ])->values()->all(),
            'action_items' => $entry->actionItems->map(fn (OneOnOneActionItem $p) => [
                'id' => (string) $p->id,
                'description' => $p->description,
                'checked' => (bool) $p->checked,
            ])->values()->all(),
            'notes' => $entry->notes->map(fn (OneOnOneNote $n) => [
                'id' => (string) $n->id,
                'note' => $n->note,
                'created_at' => $n->created_at?->toIso8601String(),
            ])->values()->all(),
            'created_at' => $entry->created_at?->toIso8601String(),
        ];
    }

    public function listOpenOneOnOnesForManager(Company $company, Employee $manager): array
    {
        return OneOnOneEntry::query()
            ->where('company_id', $company->id)
            ->where('manager_id', $manager->id)
            ->where('happened', false)
            ->with(['talkingPoints', 'actionItems', 'notes', 'manager', 'employee'])
            ->orderBy('created_at')
            ->get()
            ->map(fn (OneOnOneEntry $e) => $this->oneOnOnePayload($e))
            ->values()
            ->all();
    }

    public function listOpenOneOnOnesForEmployee(Company $company, Employee $employee): array
    {
        return OneOnOneEntry::query()
            ->where('company_id', $company->id)
            ->where('employee_id', $employee->id)
            ->where('happened', false)
            ->with(['talkingPoints', 'actionItems', 'notes', 'manager', 'employee'])
            ->orderBy('created_at')
            ->get()
            ->map(fn (OneOnOneEntry $e) => $this->oneOnOnePayload($e))
            ->values()
            ->all();
    }

    public function ensureOpenOneOnOnesForManager(Company $company, Employee $manager): void
    {
        $reportIds = DirectReport::query()
            ->where('company_id', $company->id)
            ->where('manager_id', $manager->id)
            ->pluck('employee_id');

        foreach ($reportIds as $reportId) {
            $employee = Employee::query()->find($reportId);
            if ($employee === null || $employee->locked) {
                continue;
            }
            $this->createOrGetOpenOneOnOne($company, $manager, $employee);
        }
    }

    public function ensureOpenOneOnOnesForEmployee(Company $company, Employee $employee): void
    {
        $managerIds = DirectReport::query()
            ->where('company_id', $company->id)
            ->where('employee_id', $employee->id)
            ->pluck('manager_id');

        foreach ($managerIds as $managerId) {
            $manager = Employee::query()->find($managerId);
            if ($manager === null || $manager->locked) {
                continue;
            }
            $this->createOrGetOpenOneOnOne($company, $manager, $employee);
        }
    }

    // --- Rate your manager ---

    public function pendingRateAnswers(Employee $employee): array
    {
        return RateYourManagerAnswer::query()
            ->where('employee_id', $employee->id)
            ->where('active', true)
            ->whereHas('survey', fn ($q) => $q->where('active', true))
            ->with(['survey.manager'])
            ->get()
            ->map(fn (RateYourManagerAnswer $a) => $this->rateAnswerPayload($a))
            ->values()
            ->all();
    }

    public function rateAnswerPayload(RateYourManagerAnswer $answer): array
    {
        $answer->loadMissing(['survey.manager', 'employee']);

        return [
            'id' => (string) $answer->id,
            'survey_id' => (string) $answer->rate_your_manager_survey_id,
            'employee' => $this->employeeSummary($answer->employee),
            'manager' => $answer->survey->manager
                ? $this->employeeSummary($answer->survey->manager)
                : null,
            'active' => (bool) $answer->active,
            'rating' => $answer->rating,
            'comment' => $answer->comment,
            'reveal_identity_to_manager' => (bool) $answer->reveal_identity_to_manager,
            'valid_until_at' => $answer->survey->valid_until_at?->toIso8601String(),
            'survey_active' => (bool) $answer->survey->active,
        ];
    }

    public function submitRating(Employee $actor, RateYourManagerAnswer $answer, string $rating): RateYourManagerAnswer
    {
        if ($actor->id !== $answer->employee_id) {
            throw new RuntimeException('Forbidden', 403);
        }
        if (! $answer->active || ! $answer->survey->active) {
            throw new RuntimeException('Survey is not active', 409);
        }
        if (! in_array($rating, ['bad', 'average', 'good'], true)) {
            throw new RuntimeException('Rating must be bad, average, or good', 422);
        }

        $answer->rating = $rating;
        $answer->active = false;
        $answer->save();

        return $answer->fresh(['survey.manager', 'employee']);
    }

    public function commentOnRating(
        Employee $actor,
        RateYourManagerAnswer $answer,
        ?string $comment,
        bool $revealIdentity,
    ): RateYourManagerAnswer {
        if ($actor->id !== $answer->employee_id) {
            throw new RuntimeException('Forbidden', 403);
        }
        if (! $answer->survey->active) {
            throw new RuntimeException('Survey is not active', 409);
        }

        $answer->comment = $comment;
        $answer->reveal_identity_to_manager = $revealIdentity;
        $answer->save();

        return $answer->fresh(['survey.manager', 'employee']);
    }

    public function startRateManagerSurveys(?Carbon $now = null): int
    {
        $now = $now ?? now();
        $created = 0;
        $validUntil = $now->copy()->endOfDay()->addWeekdays(3);

        $managerIds = DirectReport::query()
            ->select('manager_id', 'company_id')
            ->distinct()
            ->get();

        foreach ($managerIds as $row) {
            $manager = Employee::query()->find($row->manager_id);
            if ($manager === null || $manager->locked) {
                continue;
            }

            $survey = RateYourManagerSurvey::query()->create([
                'company_id' => $row->company_id,
                'manager_id' => $manager->id,
                'active' => true,
                'valid_until_at' => $validUntil,
            ]);

            $reports = DirectReport::query()
                ->where('company_id', $row->company_id)
                ->where('manager_id', $manager->id)
                ->pluck('employee_id');

            foreach ($reports as $reportId) {
                $employee = Employee::query()->find($reportId);
                if ($employee === null || $employee->locked) {
                    continue;
                }
                RateYourManagerAnswer::query()->create([
                    'rate_your_manager_survey_id' => $survey->id,
                    'employee_id' => $employee->id,
                    'active' => true,
                ]);
                $this->notifications->create(
                    Company::query()->findOrFail($row->company_id),
                    $employee,
                    'rate_manager.pending',
                    [
                        'survey_id' => (string) $survey->id,
                        'manager_id' => (string) $manager->id,
                    ],
                );
            }
            $created++;
        }

        return $created;
    }

    public function stopRateManagerSurveys(bool $force = false, ?Carbon $now = null): int
    {
        $now = $now ?? now();
        $query = RateYourManagerSurvey::query()->where('active', true);
        if (! $force) {
            $query->where('valid_until_at', '<=', $now);
        }

        $stopped = 0;
        $query->chunkById(50, function ($surveys) use (&$stopped) {
            foreach ($surveys as $survey) {
                $survey->active = false;
                $survey->save();
                RateYourManagerAnswer::query()
                    ->where('rate_your_manager_survey_id', $survey->id)
                    ->update(['active' => false]);
                $stopped++;
            }
        });

        return $stopped;
    }

    public function surveysForManager(Company $company, Employee $actor, Employee $manager): array
    {
        if ($actor->id !== $manager->id && ! $this->isHr($actor)) {
            throw new RuntimeException('Forbidden', 403);
        }

        return RateYourManagerSurvey::query()
            ->where('company_id', $company->id)
            ->where('manager_id', $manager->id)
            ->with(['answers.employee', 'manager'])
            ->orderByDesc('created_at')
            ->get()
            ->map(function (RateYourManagerSurvey $s) use ($actor) {
                $answers = $s->answers->map(function (RateYourManagerAnswer $a) use ($actor, $s) {
                    $showIdentity = $actor->id !== $s->manager_id
                        || $a->reveal_identity_to_manager
                        || $this->isHr($actor);

                    return [
                        'id' => (string) $a->id,
                        'rating' => $a->rating,
                        'comment' => $a->comment,
                        'reveal_identity_to_manager' => (bool) $a->reveal_identity_to_manager,
                        'employee' => $showIdentity ? $this->employeeSummary($a->employee) : null,
                    ];
                })->values()->all();

                return [
                    'id' => (string) $s->id,
                    'active' => (bool) $s->active,
                    'valid_until_at' => $s->valid_until_at?->toIso8601String(),
                    'created_at' => $s->created_at?->toIso8601String(),
                    'answers' => $answers,
                ];
            })
            ->values()
            ->all();
    }

    // --- Skills ---

    public function normalizeSkillName(string $name): string
    {
        return Str::of($name)->ascii()->lower()->trim()->toString();
    }

    public function listCompanySkills(Company $company): array
    {
        return Skill::query()
            ->where('company_id', $company->id)
            ->withCount('employees')
            ->orderBy('name')
            ->get()
            ->map(fn (Skill $s) => [
                'id' => (string) $s->id,
                'name' => $s->name,
                'employees_count' => (int) $s->employees_count,
            ])
            ->values()
            ->all();
    }

    public function skillDetail(Skill $skill): array
    {
        $skill->load('employees');

        return [
            'id' => (string) $skill->id,
            'name' => $skill->name,
            'employees' => $skill->employees->map(fn (Employee $e) => $this->employeeSummary($e))->values()->all(),
        ];
    }

    public function updateSkill(Company $company, Skill $skill, string $name): Skill
    {
        $normalized = $this->normalizeSkillName($name);
        $exists = Skill::query()
            ->where('company_id', $company->id)
            ->where('name', $normalized)
            ->where('id', '!=', $skill->id)
            ->exists();
        if ($exists) {
            throw new RuntimeException('Skill name already exists', 409);
        }
        $skill->name = $normalized;
        $skill->save();

        return $skill;
    }

    public function destroySkill(Skill $skill): void
    {
        $skill->employees()->detach();
        $skill->delete();
    }

    public function employeeSkills(Employee $employee): array
    {
        return $employee->belongsToMany(Skill::class, 'employee_skill')
            ->withTimestamps()
            ->orderBy('name')
            ->get()
            ->map(fn (Skill $s) => [
                'id' => (string) $s->id,
                'name' => $s->name,
            ])
            ->values()
            ->all();
    }

    public function attachSkill(Company $company, Employee $actor, Employee $subject, string $name): Skill
    {
        if ($actor->id !== $subject->id && ! $this->isHr($actor)) {
            throw new RuntimeException('Forbidden', 403);
        }

        $normalized = $this->normalizeSkillName($name);
        if ($normalized === '') {
            throw new RuntimeException('Skill name required', 422);
        }

        $skill = Skill::query()->firstOrCreate(
            ['company_id' => $company->id, 'name' => $normalized],
        );

        $subject->belongsToMany(Skill::class, 'employee_skill')
            ->syncWithoutDetaching([$skill->id]);

        return $skill;
    }

    public function detachSkill(Employee $actor, Employee $subject, Skill $skill): void
    {
        if ($actor->id !== $subject->id && ! $this->isHr($actor)) {
            throw new RuntimeException('Forbidden', 403);
        }

        $subject->belongsToMany(Skill::class, 'employee_skill')->detach($skill->id);

        if ($skill->employees()->count() === 0) {
            $skill->delete();
        }
    }

    public function searchSkills(Company $company, string $q): array
    {
        $normalized = $this->normalizeSkillName($q);

        return Skill::query()
            ->where('company_id', $company->id)
            ->where('name', 'like', '%'.$normalized.'%')
            ->orderBy('name')
            ->limit(20)
            ->get()
            ->map(fn (Skill $s) => [
                'id' => (string) $s->id,
                'name' => $s->name,
            ])
            ->values()
            ->all();
    }

    // --- e-Coffee ---

    public function getECoffeeEnabled(Company $company): bool
    {
        return (bool) $company->e_coffee_enabled;
    }

    public function setECoffeeEnabled(Company $company, bool $enabled): Company
    {
        $company->e_coffee_enabled = $enabled;
        $company->save();

        return $company;
    }

    public function currentECoffeeMatch(Company $company, Employee $employee): ?array
    {
        if (! $company->e_coffee_enabled) {
            return null;
        }

        $session = ECoffee::query()
            ->where('company_id', $company->id)
            ->where('active', true)
            ->first();
        if ($session === null) {
            return null;
        }

        $match = ECoffeeMatch::query()
            ->where('e_coffee_id', $session->id)
            ->where(function ($q) use ($employee) {
                $q->where('employee_id', $employee->id)
                    ->orWhere('with_employee_id', $employee->id);
            })
            ->with(['employee', 'withEmployee'])
            ->first();

        if ($match === null) {
            return null;
        }

        return $this->eCoffeeMatchPayload($match, $session);
    }

    public function eCoffeeMatchPayload(ECoffeeMatch $match, ?ECoffee $session = null): array
    {
        $match->loadMissing(['employee', 'withEmployee', 'eCoffee']);
        $session ??= $match->eCoffee;

        return [
            'id' => (string) $match->id,
            'e_coffee_id' => (string) $match->e_coffee_id,
            'batch_number' => (int) $session->batch_number,
            'employee' => $this->employeeSummary($match->employee),
            'with_employee' => $this->employeeSummary($match->withEmployee),
            'happened' => (bool) $match->happened,
        ];
    }

    public function markECoffeeHappened(Employee $actor, ECoffeeMatch $match): ECoffeeMatch
    {
        if ($actor->id !== $match->employee_id && $actor->id !== $match->with_employee_id) {
            throw new RuntimeException('Forbidden', 403);
        }
        $match->happened = true;
        $match->save();

        return $match->fresh(['employee', 'withEmployee', 'eCoffee']);
    }

    public function employeeECoffeeHistory(Company $company, Employee $actor, Employee $subject): array
    {
        if ($actor->id !== $subject->id && ! $this->isHr($actor)) {
            throw new RuntimeException('Forbidden', 403);
        }

        return ECoffeeMatch::query()
            ->where(function ($q) use ($subject) {
                $q->where('employee_id', $subject->id)
                    ->orWhere('with_employee_id', $subject->id);
            })
            ->whereHas('eCoffee', fn ($q) => $q->where('company_id', $company->id))
            ->with(['employee', 'withEmployee', 'eCoffee'])
            ->orderByDesc('created_at')
            ->limit(50)
            ->get()
            ->map(fn (ECoffeeMatch $m) => $this->eCoffeeMatchPayload($m))
            ->values()
            ->all();
    }

    public function startECoffeeSessions(): int
    {
        $started = 0;

        Company::query()
            ->where('e_coffee_enabled', true)
            ->chunkById(50, function ($companies) use (&$started) {
                foreach ($companies as $company) {
                    $this->matchEmployeesForECoffee($company);
                    $started++;
                }
            });

        return $started;
    }

    public function matchEmployeesForECoffee(Company $company): ECoffee
    {
        /** @var Collection<int, Employee> $employees */
        $employees = Employee::query()
            ->where('company_id', $company->id)
            ->where('locked', false)
            ->get()
            ->shuffle()
            ->values();

        $latest = ECoffee::query()
            ->where('company_id', $company->id)
            ->orderByDesc('batch_number')
            ->first();
        $batch = $latest ? $latest->batch_number + 1 : 1;

        ECoffee::query()
            ->where('company_id', $company->id)
            ->where('active', true)
            ->update(['active' => false]);

        $session = ECoffee::query()->create([
            'company_id' => $company->id,
            'batch_number' => $batch,
            'active' => true,
        ]);

        if ($employees->count() < 2) {
            return $session;
        }

        $ids = $employees->pluck('id')->all();
        $half = (int) floor(count($ids) / 2);
        $first = array_slice($ids, 0, $half);
        $second = array_slice($ids, $half);

        if (count($second) > count($first)) {
            // odd: reuse one from first half into second pairing side conceptually —
            // pair min(len) then leftover gets paired with someone from first
            $extra = array_pop($second);
            if ($extra !== null && count($first) > 0) {
                $second[] = $extra;
                // make lengths equal by duplicating a first-half person into second matching
                // Pattern: zip first with second; if odd count originally, one person appears twice
                while (count($second) > count($first) && count($first) > 0) {
                    $first[] = $first[array_rand($first)];
                }
            }
        }

        $pairs = min(count($first), count($second));
        for ($i = 0; $i < $pairs; $i++) {
            ECoffeeMatch::query()->create([
                'e_coffee_id' => $session->id,
                'employee_id' => $first[$i],
                'with_employee_id' => $second[$i],
                'happened' => false,
            ]);
        }

        return $session;
    }

    // --- Discipline ---

    public function listDisciplineCases(Company $company, Employee $actor, ?bool $active = null): array
    {
        $query = DisciplineCase::query()->where('company_id', $company->id);

        if ($this->isHr($actor)) {
            // all
        } else {
            $reportIds = DirectReport::query()
                ->where('company_id', $company->id)
                ->where('manager_id', $actor->id)
                ->pluck('employee_id');
            $query->whereIn('employee_id', $reportIds);
        }

        if ($active !== null) {
            $query->where('active', $active);
        }

        return $query->with(['employee', 'openedBy'])
            ->orderByDesc('created_at')
            ->get()
            ->map(fn (DisciplineCase $c) => $this->disciplineCasePayload($c, false))
            ->values()
            ->all();
    }

    public function ensureDisciplineAccess(Employee $actor, DisciplineCase $case): void
    {
        if ($this->isHr($actor)) {
            return;
        }
        $employee = Employee::query()->findOrFail($case->employee_id);
        if ($this->isManagerOf($actor, $employee)) {
            return;
        }
        throw new RuntimeException('Forbidden', 403);
    }

    public function createDisciplineCase(Company $company, Employee $actor, Employee $subject): DisciplineCase
    {
        if (! $this->isHr($actor) && ! $this->isManagerOf($actor, $subject)) {
            throw new RuntimeException('Forbidden', 403);
        }

        return DisciplineCase::query()->create([
            'company_id' => $company->id,
            'employee_id' => $subject->id,
            'opened_by_employee_id' => $actor->id,
            'opened_by_employee_name' => trim($actor->first_name.' '.$actor->last_name),
            'active' => true,
        ])->load(['employee', 'openedBy', 'events']);
    }

    public function toggleDisciplineCase(Employee $actor, DisciplineCase $case): DisciplineCase
    {
        if (! $this->isHr($actor)) {
            throw new RuntimeException('Forbidden', 403);
        }
        $case->active = ! $case->active;
        $case->save();

        return $case->fresh(['employee', 'openedBy', 'events.author']);
    }

    public function destroyDisciplineCase(Employee $actor, DisciplineCase $case): void
    {
        if (! $this->isHr($actor)) {
            throw new RuntimeException('Forbidden', 403);
        }
        $case->delete();
    }

    public function createDisciplineEvent(
        Employee $actor,
        DisciplineCase $case,
        string $happenedAt,
        string $description,
    ): DisciplineEvent {
        $this->ensureDisciplineAccess($actor, $case);

        return DisciplineEvent::query()->create([
            'discipline_case_id' => $case->id,
            'author_id' => $actor->id,
            'author_name' => trim($actor->first_name.' '.$actor->last_name),
            'happened_at' => $happenedAt,
            'description' => $description,
        ]);
    }

    public function destroyDisciplineEvent(Employee $actor, DisciplineEvent $event): void
    {
        $case = $event->disciplineCase;
        $this->ensureDisciplineAccess($actor, $case);
        $event->delete();
    }

    public function attachDisciplineFile(
        Employee $actor,
        DisciplineEvent $event,
        int $temporaryUploadId,
        int $mediaId,
    ): Media {
        $this->ensureDisciplineAccess($actor, $event->disciplineCase);

        return $this->mediaAttach->attachFromTemporary(
            $event,
            'discipline',
            $temporaryUploadId,
            $mediaId,
            clearExisting: false,
        );
    }

    public function listDisciplineFiles(DisciplineEvent $event): array
    {
        return $event->getMedia('discipline')->map(fn (Media $m) => [
            'id' => $m->id,
            'file_name' => $m->file_name,
            'mime_type' => $m->mime_type,
            'size' => $m->size,
            'url' => url('/api/v1/media/'.$m->id.'/file'),
        ])->values()->all();
    }

    public function disciplineCasePayload(DisciplineCase $case, bool $withEvents = true): array
    {
        $case->loadMissing(['employee', 'openedBy']);
        if ($withEvents) {
            $case->loadMissing(['events.author']);
        }

        $payload = [
            'id' => (string) $case->id,
            'employee' => $this->employeeSummary($case->employee),
            'opened_by' => $case->openedBy
                ? $this->employeeSummary($case->openedBy)
                : null,
            'opened_by_employee_name' => $case->opened_by_employee_name,
            'active' => (bool) $case->active,
            'created_at' => $case->created_at?->toIso8601String(),
        ];

        if ($withEvents) {
            $payload['events'] = $case->events->map(fn (DisciplineEvent $e) => [
                'id' => (string) $e->id,
                'author_name' => $e->author_name,
                'author' => $e->author ? $this->employeeSummary($e->author) : null,
                'happened_at' => $e->happened_at?->toDateString(),
                'description' => $e->description,
                'files' => $this->listDisciplineFiles($e),
                'created_at' => $e->created_at?->toIso8601String(),
            ])->values()->all();
        }

        return $payload;
    }

    public function activeDisciplineCount(Company $company, Employee $actor): int
    {
        $query = DisciplineCase::query()
            ->where('company_id', $company->id)
            ->where('active', true);

        if (! $this->isHr($actor)) {
            $reportIds = DirectReport::query()
                ->where('company_id', $company->id)
                ->where('manager_id', $actor->id)
                ->pluck('employee_id');
            $query->whereIn('employee_id', $reportIds);
        }

        return $query->count();
    }
}
