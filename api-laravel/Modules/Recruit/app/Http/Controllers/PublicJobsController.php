<?php

namespace Modules\Recruit\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Recruit\Services\RecruitService;
use RuntimeException;
use Spatie\MediaLibrary\MediaCollections\Models\Media;

class PublicJobsController extends Controller
{
    public function __construct(private readonly RecruitService $recruit) {}

    public function listCompanies(): JsonResponse
    {
        return ApiResponse::success($this->recruit->listPublicCompanies());
    }

    public function listCompanyJobs(string $companySlug): JsonResponse
    {
        $company = $this->findCompany($companySlug);

        return ApiResponse::success($this->recruit->listPublicOpenings($company));
    }

    public function showJob(string $companySlug, string $jobSlug): JsonResponse
    {
        $company = $this->findCompany($companySlug);

        try {
            $job = $this->recruit->showPublicOpening($company, $jobSlug);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($job);
    }

    public function apply(Request $request, string $companySlug, string $jobSlug): JsonResponse
    {
        $company = $this->findCompany($companySlug);
        $data = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'email' => ['required', 'email', 'max:255'],
            'url' => ['nullable', 'string', 'max:2048'],
            'desired_salary' => ['nullable', 'string', 'max:255'],
            'notes' => ['nullable', 'string'],
        ]);

        try {
            $candidate = $this->recruit->createPublicCandidate($company, $jobSlug, $data);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($candidate, 'Application started', 201);
    }

    public function listFiles(string $companySlug, string $jobSlug, string $candidateUuid): JsonResponse
    {
        try {
            $company = $this->findCompany($companySlug);
            $candidate = $this->recruit->findIncompleteCandidate($company, $jobSlug, $candidateUuid);
            $files = $this->recruit->listFiles($candidate);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($files);
    }

    public function attachFile(
        Request $request,
        string $companySlug,
        string $jobSlug,
        string $candidateUuid,
    ): JsonResponse {
        $data = $request->validate([
            'temporary_upload_id' => ['required', 'integer', 'exists:temporary_uploads,id'],
            'media_id' => ['required', 'integer', 'exists:media,id'],
        ]);

        try {
            $company = $this->findCompany($companySlug);
            $candidate = $this->recruit->findIncompleteCandidate($company, $jobSlug, $candidateUuid);
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
        string $companySlug,
        string $jobSlug,
        string $candidateUuid,
        int $mediaId,
    ): JsonResponse {
        try {
            $company = $this->findCompany($companySlug);
            $candidate = $this->recruit->findIncompleteCandidate($company, $jobSlug, $candidateUuid);
            $this->recruit->deleteFile($candidate, $mediaId);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'File deleted');
    }

    public function complete(string $companySlug, string $jobSlug, string $candidateUuid): JsonResponse
    {
        try {
            $company = $this->findCompany($companySlug);
            $candidate = $this->recruit->findIncompleteCandidate($company, $jobSlug, $candidateUuid);
            $result = $this->recruit->completeApplication($candidate);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success($result, 'Application completed');
    }

    public function abandon(string $companySlug, string $jobSlug, string $candidateUuid): JsonResponse
    {
        try {
            $company = $this->findCompany($companySlug);
            $candidate = $this->recruit->findIncompleteCandidate($company, $jobSlug, $candidateUuid);
            $this->recruit->abandonApplication($candidate);
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(null, 'Application abandoned');
    }

    private function findCompany(string $slug): Company
    {
        return Company::query()->where('slug', $slug)->firstOrFail();
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
