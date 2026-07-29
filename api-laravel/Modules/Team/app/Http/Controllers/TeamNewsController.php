<?php

namespace Modules\Team\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Employee\Models\Employee;
use Modules\Team\Models\Team;
use Modules\Team\Models\TeamNews;
use Modules\Team\Services\CommunicateService;

class TeamNewsController extends Controller
{
    public function __construct(private readonly CommunicateService $communicate) {}

    public function index(Request $request, string $companyId, string $teamId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $team = $this->findTeam($request, $teamId);

        if (! $this->communicate->canAccessTeam($actor, $team, 'team-news.view')) {
            return ApiResponse::error('Forbidden', 403);
        }

        $list = TeamNews::query()
            ->where('team_id', $team->id)
            ->orderByDesc('created_at')
            ->get()
            ->map(fn (TeamNews $n) => $this->communicate->teamNewsPayload($n))
            ->values()
            ->all();

        return ApiResponse::success($list);
    }

    public function store(Request $request, string $companyId, string $teamId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $team = $this->findTeam($request, $teamId);

        if (! $this->communicate->canAccessTeam($actor, $team, 'team-news.create')) {
            return ApiResponse::error('Forbidden', 403);
        }

        $validated = $request->validate([
            'title' => ['required', 'string', 'max:255'],
            'content' => ['required', 'string'],
        ]);

        $news = $this->communicate->createTeamNews($team, $actor, $validated);

        return ApiResponse::success($this->communicate->teamNewsPayload($news), 'Team news created', 201);
    }

    public function show(Request $request, string $companyId, string $teamId, string $newsId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $team = $this->findTeam($request, $teamId);

        if (! $this->communicate->canAccessTeam($actor, $team, 'team-news.view')) {
            return ApiResponse::error('Forbidden', 403);
        }

        $news = $this->findNews($team, $newsId);

        return ApiResponse::success($this->communicate->teamNewsPayload($news));
    }

    public function update(Request $request, string $companyId, string $teamId, string $newsId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $team = $this->findTeam($request, $teamId);
        $news = $this->findNews($team, $newsId);

        if (! $this->communicate->canManageTeamNews($actor, $news, 'team-news.update')) {
            return ApiResponse::error('Forbidden', 403);
        }

        $validated = $request->validate([
            'title' => ['sometimes', 'string', 'max:255'],
            'content' => ['sometimes', 'string'],
        ]);

        $updated = $this->communicate->updateTeamNews($news, $validated, $actor);

        return ApiResponse::success($this->communicate->teamNewsPayload($updated), 'Team news updated');
    }

    public function destroy(Request $request, string $companyId, string $teamId, string $newsId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $team = $this->findTeam($request, $teamId);
        $news = $this->findNews($team, $newsId);

        if (! $this->communicate->canManageTeamNews($actor, $news, 'team-news.delete')) {
            return ApiResponse::error('Forbidden', 403);
        }

        $this->communicate->destroyTeamNews($news, $actor);

        return ApiResponse::success(null, 'Team news deleted');
    }

    private function findTeam(Request $request, string $teamId): Team
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return Team::query()
            ->where('company_id', $company->id)
            ->where('id', $teamId)
            ->firstOrFail();
    }

    private function findNews(Team $team, string $newsId): TeamNews
    {
        return TeamNews::query()
            ->where('team_id', $team->id)
            ->where('id', $newsId)
            ->firstOrFail();
    }
}
