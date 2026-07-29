<?php

namespace Modules\Company\Http\Controllers;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Modules\Company\Models\Company;
use Modules\Company\Services\CompanyService;
use Modules\Uploads\Services\MediaAttachService;
use RuntimeException;

class CompanyLogoController extends Controller
{
    public function __construct(
        private readonly CompanyService $companies,
        private readonly MediaAttachService $mediaAttach,
    ) {}

    public function update(Request $request): JsonResponse
    {
        /** @var Company $company */
        $company = $request->attributes->get('company');

        $validated = $request->validate([
            'temporary_upload_id' => ['required', 'integer', 'exists:temporary_uploads,id'],
            'media_id' => ['required', 'integer', 'exists:media,id'],
        ]);

        try {
            $this->mediaAttach->attachFromTemporary(
                $company,
                'logo',
                (int) $validated['temporary_upload_id'],
                (int) $validated['media_id'],
            );
        } catch (RuntimeException $e) {
            return ApiResponse::error($e->getMessage(), $e->getCode() ?: 400);
        }

        return ApiResponse::success(
            $this->companies->companyPayload($company->fresh(), includeJoinCode: true),
            'Logo updated',
        );
    }
}
