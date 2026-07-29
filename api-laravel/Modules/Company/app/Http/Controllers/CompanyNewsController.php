<?php

namespace Modules\Company\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Company\Models\CompanyNews;
use Modules\Company\Services\CompanyNewsService;
use Modules\Employee\Models\Employee;

class CompanyNewsController extends Controller
{
    public function __construct(private readonly CompanyNewsService $news) {}

    public function index(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        $list = CompanyNews::query()
            ->where('company_id', $company->id)
            ->orderByDesc('created_at')
            ->get()
            ->map(fn (CompanyNews $n) => $this->news->payload($n))
            ->values()
            ->all();

        return ApiResponse::success($list);
    }

    public function show(Request $request, string $companyId, string $newsId): JsonResponse
    {
        $item = $this->findInCompany($request, $newsId);

        return ApiResponse::success($this->news->payload($item));
    }

    public function store(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');

        $validated = $request->validate([
            'title' => ['required', 'string', 'max:255'],
            'content' => ['required', 'string'],
        ]);

        $item = $this->news->create($company, $actor, $validated);

        return ApiResponse::success($this->news->payload($item), 'News created', 201);
    }

    public function update(Request $request, string $companyId, string $newsId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $item = $this->findInCompany($request, $newsId);

        $validated = $request->validate([
            'title' => ['sometimes', 'string', 'max:255'],
            'content' => ['sometimes', 'string'],
        ]);

        $updated = $this->news->update($item, $validated, $actor);

        return ApiResponse::success($this->news->payload($updated), 'News updated');
    }

    public function destroy(Request $request, string $companyId, string $newsId): JsonResponse
    {
        /** @var Employee $actor */
        $actor = $request->attributes->get('employee');
        $item = $this->findInCompany($request, $newsId);
        $this->news->destroy($item, $actor);

        return ApiResponse::success(null, 'News deleted');
    }

    private function findInCompany(Request $request, string $newsId): CompanyNews
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        return CompanyNews::query()
            ->where('company_id', $company->id)
            ->where('id', $newsId)
            ->firstOrFail();
    }
}
