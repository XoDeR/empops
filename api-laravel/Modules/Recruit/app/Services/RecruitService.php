<?php

namespace Modules\Recruit\Services;

use Carbon\Carbon;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Str;
use Modules\Company\Models\Company;
use Modules\Company\Services\FlowService;
use Modules\Employee\Models\Employee;
use Modules\Employee\Services\EmployeeService;
use Modules\Recruit\Models\Candidate;
use Modules\Recruit\Models\CandidateStage;
use Modules\Recruit\Models\CandidateStageNote;
use Modules\Recruit\Models\CandidateStageParticipant;
use Modules\Recruit\Models\JobOpening;
use Modules\Recruit\Models\RecruitingStage;
use Modules\Recruit\Models\RecruitingStageTemplate;
use Modules\Team\Models\Team;
use Modules\Team\Services\TeamService;
use Modules\Uploads\Services\MediaAttachService;
use RuntimeException;
use Spatie\MediaLibrary\MediaCollections\Models\Media;

final class RecruitService
{
    public function __construct(
        private readonly MediaAttachService $mediaAttach,
        private readonly EmployeeService $employees,
        private readonly TeamService $teams,
    ) {}

    public function canManageRecruiting(Employee $actor): bool
    {
        return $actor->can('recruiting.view')
            || $actor->can('recruiting.create')
            || $actor->can('recruiting.update')
            || $actor->hasAnyRole(['administrator', 'hr']);
    }

    public function isSponsor(Employee $actor, JobOpening $opening): bool
    {
        return $opening->sponsors()->where('employees.id', $actor->id)->exists();
    }

    public function canAccessOpening(Employee $actor, JobOpening $opening): bool
    {
        return $this->canManageRecruiting($actor) || $this->isSponsor($actor, $opening);
    }

    public function ensureAccess(Employee $actor, JobOpening $opening): void
    {
        if (! $this->canAccessOpening($actor, $opening)) {
            throw new RuntimeException('Forbidden', 403);
        }
    }

    public function ensureManage(Employee $actor): void
    {
        if (! $actor->can('recruiting.update') && ! $actor->hasAnyRole(['administrator', 'hr'])) {
            throw new RuntimeException('Forbidden', 403);
        }
    }

    // --- Templates ---

    public function listTemplates(Company $company): Collection
    {
        return RecruitingStageTemplate::query()
            ->with('stages')
            ->where('company_id', $company->id)
            ->orderBy('name')
            ->get()
            ->map(fn (RecruitingStageTemplate $t) => $this->templatePayload($t));
    }

    public function createTemplate(Company $company, string $name): array
    {
        $template = RecruitingStageTemplate::query()->create([
            'company_id' => $company->id,
            'name' => $name,
        ]);

        return $this->templatePayload($template->load('stages'));
    }

    public function updateTemplate(RecruitingStageTemplate $template, string $name): array
    {
        $template->name = $name;
        $template->save();

        return $this->templatePayload($template->fresh('stages'));
    }

    public function deleteTemplate(RecruitingStageTemplate $template): void
    {
        $template->delete();
    }

    public function createStage(RecruitingStageTemplate $template, string $name): array
    {
        $max = (int) $template->stages()->max('position');
        $stage = RecruitingStage::query()->create([
            'recruiting_stage_template_id' => $template->id,
            'name' => $name,
            'position' => $max + 1,
        ]);

        return $this->stagePayload($stage);
    }

    public function updateStage(RecruitingStage $stage, array $data): array
    {
        if (isset($data['name'])) {
            $stage->name = $data['name'];
        }

        if (isset($data['position'])) {
            $newPos = (int) $data['position'];
            $oldPos = (int) $stage->position;
            if ($newPos !== $oldPos) {
                $siblings = RecruitingStage::query()
                    ->where('recruiting_stage_template_id', $stage->recruiting_stage_template_id)
                    ->where('id', '!=', $stage->id)
                    ->orderBy('position')
                    ->get();

                $ordered = $siblings->values()->all();
                array_splice($ordered, max(0, min($newPos, count($ordered))), 0, [$stage]);
                foreach ($ordered as $i => $s) {
                    $s->position = $i;
                    $s->save();
                }
            }
        }

        $stage->save();

        return $this->stagePayload($stage->fresh());
    }

    public function deleteStage(RecruitingStage $stage): void
    {
        $templateId = $stage->recruiting_stage_template_id;
        $stage->delete();
        $remaining = RecruitingStage::query()
            ->where('recruiting_stage_template_id', $templateId)
            ->orderBy('position')
            ->get();
        foreach ($remaining as $i => $s) {
            $s->position = $i;
            $s->save();
        }
    }

    // --- Job openings ---

    public function listOpenings(Company $company, Employee $actor, ?bool $fulfilled = null): Collection
    {
        $query = JobOpening::query()
            ->with(['sponsors', 'position', 'template', 'team'])
            ->where('company_id', $company->id);

        if ($fulfilled === true) {
            $query->where('fulfilled', true);
        } elseif ($fulfilled === false) {
            $query->where('fulfilled', false);
        }

        if (! $this->canManageRecruiting($actor)) {
            $query->whereHas('sponsors', fn ($q) => $q->where('employees.id', $actor->id));
        }

        return $query->orderByDesc('created_at')
            ->get()
            ->map(fn (JobOpening $o) => $this->openingPayload($o));
    }

    public function createOpening(Company $company, array $data): array
    {
        return DB::transaction(function () use ($company, $data) {
            $opening = JobOpening::query()->create([
                'company_id' => $company->id,
                'position_id' => $data['position_id'],
                'recruiting_stage_template_id' => $data['recruiting_stage_template_id'],
                'team_id' => $data['team_id'] ?? null,
                'title' => $data['title'],
                'description' => $data['description'],
                'slug' => $this->uniqueJobSlug($company, $data['title']),
                'reference_number' => $data['reference_number'] ?? null,
                'active' => false,
                'fulfilled' => false,
                'page_views' => 0,
            ]);

            if (! empty($data['sponsor_ids'])) {
                $opening->sponsors()->sync($data['sponsor_ids']);
            }

            return $this->openingPayload($opening->fresh(['sponsors', 'position', 'template', 'team']));
        });
    }

    public function updateOpening(JobOpening $opening, array $data): array
    {
        $opening->fill(collect($data)->only([
            'title',
            'description',
            'position_id',
            'recruiting_stage_template_id',
            'team_id',
            'reference_number',
        ])->all());

        if (isset($data['title']) && $data['title'] !== $opening->getOriginal('title')) {
            $opening->slug = $this->uniqueJobSlug($opening->company, $data['title'], (string) $opening->id);
        }

        $opening->save();

        if (array_key_exists('sponsor_ids', $data)) {
            $opening->sponsors()->sync($data['sponsor_ids'] ?? []);
        }

        return $this->openingPayload($opening->fresh(['sponsors', 'position', 'template', 'team']));
    }

    public function deleteOpening(JobOpening $opening): void
    {
        $opening->delete();
    }

    public function toggleOpening(JobOpening $opening): array
    {
        $opening->active = ! $opening->active;
        if ($opening->active && $opening->activated_at === null) {
            $opening->activated_at = now();
        }
        $opening->save();

        return $this->openingPayload($opening->fresh(['sponsors', 'position', 'template', 'team']));
    }

    public function addSponsor(JobOpening $opening, Employee $employee): array
    {
        $opening->sponsors()->syncWithoutDetaching([$employee->id]);

        return $this->openingPayload($opening->fresh(['sponsors', 'position', 'template', 'team']));
    }

    public function removeSponsor(JobOpening $opening, Employee $employee): array
    {
        $opening->sponsors()->detach($employee->id);

        return $this->openingPayload($opening->fresh(['sponsors', 'position', 'template', 'team']));
    }

    // --- Candidates ---

    public function listCandidates(JobOpening $opening, string $bucket = 'to_sort'): Collection
    {
        $query = Candidate::query()
            ->where('job_opening_id', $opening->id)
            ->where('application_completed', true);

        if ($bucket === 'rejected') {
            $query->where('rejected', true);
        } elseif ($bucket === 'selected') {
            $query->where('rejected', false)
                ->whereHas('stages', fn ($q) => $q->where('status', '!=', 'pending'));
        } else {
            $query->where('rejected', false)
                ->whereDoesntHave('stages', fn ($q) => $q->where('status', '!=', 'pending'));
        }

        return $query->orderByDesc('created_at')
            ->get()
            ->map(fn (Candidate $c) => $this->candidatePayload($c));
    }

    public function showCandidate(Candidate $candidate): array
    {
        $candidate->load(['stages.notes', 'stages.participants']);

        return $this->candidatePayload($candidate, true);
    }

    public function processStage(CandidateStage $stage, Employee $actor, bool $accepted): array
    {
        $candidate = $stage->candidate;

        if ($candidate->rejected) {
            throw new RuntimeException('Candidate already rejected', 409);
        }

        if ($stage->status !== 'pending') {
            throw new RuntimeException('Stage already processed', 409);
        }

        $name = trim($actor->first_name.' '.$actor->last_name);

        if ($accepted) {
            $stage->status = 'passed';
        } else {
            $stage->status = 'rejected';
            $candidate->rejected = true;
            $candidate->save();
        }

        $stage->decider_id = $actor->id;
        $stage->decider_name = $name;
        $stage->decided_at = now();
        $stage->save();

        return $this->showCandidate($candidate->fresh());
    }

    public function createNote(CandidateStage $stage, Employee $actor, string $note): array
    {
        $row = CandidateStageNote::query()->create([
            'candidate_stage_id' => $stage->id,
            'author_id' => $actor->id,
            'author_name' => trim($actor->first_name.' '.$actor->last_name),
            'note' => $note,
        ]);

        return [
            'id' => $row->id,
            'author_id' => $row->author_id,
            'author_name' => $row->author_name,
            'note' => $row->note,
            'created_at' => $row->created_at?->toIso8601String(),
        ];
    }

    public function updateNote(CandidateStageNote $note, string $text): array
    {
        $note->note = $text;
        $note->save();

        return [
            'id' => $note->id,
            'author_id' => $note->author_id,
            'author_name' => $note->author_name,
            'note' => $note->note,
            'created_at' => $note->created_at?->toIso8601String(),
        ];
    }

    public function deleteNote(CandidateStageNote $note): void
    {
        $note->delete();
    }

    public function addParticipant(CandidateStage $stage, Employee $participant): array
    {
        $row = CandidateStageParticipant::query()->updateOrCreate(
            [
                'candidate_stage_id' => $stage->id,
                'participant_id' => $participant->id,
            ],
            [
                'participant_name' => trim($participant->first_name.' '.$participant->last_name),
                'participated' => false,
            ],
        );

        return [
            'id' => $row->id,
            'participant_id' => $row->participant_id,
            'participant_name' => $row->participant_name,
            'participated' => $row->participated,
        ];
    }

    public function removeParticipant(CandidateStageParticipant $participant): void
    {
        $participant->delete();
    }

    public function hire(Company $company, JobOpening $opening, Candidate $candidate, array $data, Employee $actor): array
    {
        if (! $actor->can('recruiting.hire') && ! $actor->hasAnyRole(['administrator', 'hr'])) {
            throw new RuntimeException('Forbidden', 403);
        }

        if (! $opening->active || $opening->fulfilled) {
            throw new RuntimeException('Opening is not open for hiring', 409);
        }

        if ($candidate->job_opening_id !== $opening->id) {
            throw new RuntimeException('Candidate not on this opening', 404);
        }

        if ($candidate->employee_id !== null) {
            throw new RuntimeException('Candidate already hired', 409);
        }

        return DB::transaction(function () use ($company, $opening, $candidate, $data) {
            $employee = $this->employees->create($company, [
                'email' => $data['email'],
                'first_name' => $data['first_name'],
                'last_name' => $data['last_name'],
                'hired_at' => $data['hired_at'],
                'position_id' => $opening->position_id,
            ], 'employee');

            if ($opening->team_id) {
                $team = Team::query()
                    ->where('company_id', $company->id)
                    ->where('id', $opening->team_id)
                    ->first();
                if ($team !== null) {
                    $this->teams->addMember($team, $employee);
                }
            }

            $opening->active = false;
            $opening->fulfilled = true;
            $opening->fulfilled_at = now();
            $opening->fulfilled_by_candidate_id = $candidate->id;
            $opening->save();

            $candidate->employee_id = $employee->id;
            $candidate->employee_name = trim($employee->first_name.' '.$employee->last_name);
            $candidate->save();

            app(FlowService::class)->scheduleForEmployee(
                $company,
                $employee,
                'employee_joins_company',
                Carbon::parse($data['hired_at']),
            );

            return [
                'candidate' => $this->showCandidate($candidate->fresh()),
                'employee' => $this->employees->employeePayload($employee),
                'opening' => $this->openingPayload($opening->fresh(['sponsors', 'position', 'template', 'team'])),
            ];
        });
    }

    public function attachFile(Candidate $candidate, int $temporaryUploadId, int $mediaId): Media
    {
        return $this->mediaAttach->attachFromTemporary(
            $candidate,
            'cv',
            $temporaryUploadId,
            $mediaId,
            clearExisting: false,
        );
    }

    public function listFiles(Candidate $candidate): array
    {
        return $candidate->getMedia('cv')->map(fn (Media $m) => [
            'id' => $m->id,
            'file_name' => $m->file_name,
            'mime_type' => $m->mime_type,
            'size' => $m->size,
            'url' => url('/api/v1/media/'.$m->id.'/file'),
        ])->values()->all();
    }

    public function deleteFile(Candidate $candidate, int $mediaId): void
    {
        $media = $candidate->getMedia('cv')->firstWhere('id', $mediaId);
        if ($media === null) {
            throw new RuntimeException('File not found', 404);
        }
        $media->delete();
    }

    // --- Public careers ---

    public function listPublicCompanies(): array
    {
        return Company::query()
            ->whereHas('jobOpeningsPublic')
            ->orderBy('name')
            ->get()
            ->map(fn (Company $c) => [
                'slug' => $c->slug,
                'name' => $c->name,
                'openings_count' => $c->jobOpeningsPublic()->count(),
            ])
            ->values()
            ->all();
    }

    public function listPublicOpenings(Company $company): array
    {
        return JobOpening::query()
            ->where('company_id', $company->id)
            ->where('active', true)
            ->where('fulfilled', false)
            ->orderBy('title')
            ->get()
            ->map(fn (JobOpening $o) => [
                'title' => $o->title,
                'slug' => $o->slug,
                'reference_number' => $o->reference_number,
            ])
            ->values()
            ->all();
    }

    public function showPublicOpening(Company $company, string $jobSlug): array
    {
        $opening = JobOpening::query()
            ->where('company_id', $company->id)
            ->where('slug', $jobSlug)
            ->where('active', true)
            ->where('fulfilled', false)
            ->first();

        if ($opening === null) {
            throw new RuntimeException('Job opening not found', 404);
        }

        $opening->page_views = $opening->page_views + 1;
        $opening->save();

        return [
            'title' => $opening->title,
            'slug' => $opening->slug,
            'description' => $opening->description,
            'reference_number' => $opening->reference_number,
            'company' => [
                'slug' => $company->slug,
                'name' => $company->name,
            ],
        ];
    }

    public function createPublicCandidate(Company $company, string $jobSlug, array $data): array
    {
        $opening = JobOpening::query()
            ->with('template.stages')
            ->where('company_id', $company->id)
            ->where('slug', $jobSlug)
            ->where('active', true)
            ->where('fulfilled', false)
            ->first();

        if ($opening === null) {
            throw new RuntimeException('Job opening not found', 404);
        }

        if ($opening->recruiting_stage_template_id === null || $opening->template === null) {
            throw new RuntimeException('Opening has no stage template', 422);
        }

        return DB::transaction(function () use ($company, $opening, $data) {
            $candidate = Candidate::query()->create([
                'company_id' => $company->id,
                'job_opening_id' => $opening->id,
                'name' => $data['name'],
                'email' => $data['email'],
                'uuid' => (string) Str::uuid(),
                'url' => $data['url'] ?? null,
                'desired_salary' => $data['desired_salary'] ?? null,
                'notes' => $data['notes'] ?? null,
                'application_completed' => false,
                'rejected' => false,
            ]);

            foreach ($opening->template->stages as $stage) {
                CandidateStage::query()->create([
                    'candidate_id' => $candidate->id,
                    'stage_name' => $stage->name,
                    'stage_position' => $stage->position,
                    'status' => 'pending',
                ]);
            }

            return [
                'uuid' => $candidate->uuid,
                'name' => $candidate->name,
                'email' => $candidate->email,
            ];
        });
    }

    public function findIncompleteCandidate(Company $company, string $jobSlug, string $candidateUuid): Candidate
    {
        $opening = JobOpening::query()
            ->where('company_id', $company->id)
            ->where('slug', $jobSlug)
            ->first();

        if ($opening === null) {
            throw new RuntimeException('Job opening not found', 404);
        }

        $candidate = Candidate::query()
            ->where('job_opening_id', $opening->id)
            ->where('uuid', $candidateUuid)
            ->where('application_completed', false)
            ->first();

        if ($candidate === null) {
            throw new RuntimeException('Application not found', 404);
        }

        return $candidate;
    }

    public function completeApplication(Candidate $candidate): array
    {
        $candidate->application_completed = true;
        $candidate->save();

        return [
            'uuid' => $candidate->uuid,
            'application_completed' => true,
        ];
    }

    public function abandonApplication(Candidate $candidate): void
    {
        if ($candidate->application_completed) {
            throw new RuntimeException('Cannot abandon completed application', 409);
        }
        $candidate->delete();
    }

    // --- Payloads ---

    public function templatePayload(RecruitingStageTemplate $template): array
    {
        return [
            'id' => $template->id,
            'company_id' => $template->company_id,
            'name' => $template->name,
            'stages' => $template->stages->map(fn (RecruitingStage $s) => $this->stagePayload($s))->values()->all(),
        ];
    }

    public function stagePayload(RecruitingStage $stage): array
    {
        return [
            'id' => $stage->id,
            'name' => $stage->name,
            'position' => $stage->position,
        ];
    }

    public function openingPayload(JobOpening $opening): array
    {
        return [
            'id' => $opening->id,
            'company_id' => $opening->company_id,
            'title' => $opening->title,
            'description' => $opening->description,
            'slug' => $opening->slug,
            'reference_number' => $opening->reference_number,
            'position_id' => $opening->position_id,
            'position' => $opening->position ? [
                'id' => $opening->position->id,
                'title' => $opening->position->title,
            ] : null,
            'recruiting_stage_template_id' => $opening->recruiting_stage_template_id,
            'team_id' => $opening->team_id,
            'active' => $opening->active,
            'fulfilled' => $opening->fulfilled,
            'page_views' => $opening->page_views,
            'activated_at' => $opening->activated_at?->toIso8601String(),
            'fulfilled_at' => $opening->fulfilled_at?->toIso8601String(),
            'sponsors' => $opening->sponsors->map(fn (Employee $e) => [
                'id' => $e->id,
                'first_name' => $e->first_name,
                'last_name' => $e->last_name,
            ])->values()->all(),
        ];
    }

    public function candidatePayload(Candidate $candidate, bool $detailed = false): array
    {
        $payload = [
            'id' => $candidate->id,
            'job_opening_id' => $candidate->job_opening_id,
            'name' => $candidate->name,
            'email' => $candidate->email,
            'uuid' => $candidate->uuid,
            'url' => $candidate->url,
            'desired_salary' => $candidate->desired_salary,
            'notes' => $candidate->notes,
            'application_completed' => $candidate->application_completed,
            'rejected' => $candidate->rejected,
            'employee_id' => $candidate->employee_id,
            'employee_name' => $candidate->employee_name,
            'created_at' => $candidate->created_at?->toIso8601String(),
        ];

        if ($detailed) {
            $payload['stages'] = $candidate->stages->map(function (CandidateStage $s) {
                return [
                    'id' => $s->id,
                    'stage_name' => $s->stage_name,
                    'stage_position' => $s->stage_position,
                    'status' => $s->status,
                    'decider_id' => $s->decider_id,
                    'decider_name' => $s->decider_name,
                    'decided_at' => $s->decided_at?->toIso8601String(),
                    'notes' => $s->notes->map(fn (CandidateStageNote $n) => [
                        'id' => $n->id,
                        'author_id' => $n->author_id,
                        'author_name' => $n->author_name,
                        'note' => $n->note,
                        'created_at' => $n->created_at?->toIso8601String(),
                    ])->values()->all(),
                    'participants' => $s->participants->map(fn (CandidateStageParticipant $p) => [
                        'id' => $p->id,
                        'participant_id' => $p->participant_id,
                        'participant_name' => $p->participant_name,
                        'participated' => $p->participated,
                    ])->values()->all(),
                ];
            })->values()->all();
            $payload['files'] = $this->listFiles($candidate);
        }

        return $payload;
    }

    private function uniqueJobSlug(Company $company, string $title, ?string $ignoreId = null): string
    {
        $base = Str::slug($title) ?: 'job';
        $slug = $base.'-'.Str::lower(Str::random(8));
        $i = 0;
        while (
            JobOpening::query()
                ->where('company_id', $company->id)
                ->where('slug', $slug)
                ->when($ignoreId, fn ($q) => $q->where('id', '!=', $ignoreId))
                ->exists()
        ) {
            $i++;
            $slug = $base.'-'.Str::lower(Str::random(8)).($i > 0 ? '-'.$i : '');
        }

        return $slug;
    }
}
