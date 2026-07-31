<?php

use Illuminate\Support\Facades\Route;
use Modules\Auth\Http\Middleware\AuthenticateJwt;
use Modules\Company\Http\Middleware\EnsureCompanyMember;
use Modules\Company\Http\Middleware\EnsurePermission;
use Modules\Recruit\Http\Controllers\PublicJobsController;
use Modules\Recruit\Http\Controllers\RecruitController;

Route::prefix('v1/companies/{companyId}')
    ->middleware([AuthenticateJwt::class, EnsureCompanyMember::class])
    ->group(function () {
        Route::get('recruiting/stage-templates', [RecruitController::class, 'listTemplates'])
            ->middleware(EnsurePermission::class.':recruiting.manage_templates');
        Route::post('recruiting/stage-templates', [RecruitController::class, 'storeTemplate'])
            ->middleware(EnsurePermission::class.':recruiting.manage_templates');
        Route::get('recruiting/stage-templates/{templateId}', [RecruitController::class, 'showTemplate'])
            ->middleware(EnsurePermission::class.':recruiting.manage_templates');
        Route::patch('recruiting/stage-templates/{templateId}', [RecruitController::class, 'updateTemplate'])
            ->middleware(EnsurePermission::class.':recruiting.manage_templates');
        Route::delete('recruiting/stage-templates/{templateId}', [RecruitController::class, 'destroyTemplate'])
            ->middleware(EnsurePermission::class.':recruiting.manage_templates');

        Route::post('recruiting/stage-templates/{templateId}/stages', [RecruitController::class, 'storeStage'])
            ->middleware(EnsurePermission::class.':recruiting.manage_templates');
        Route::patch('recruiting/stage-templates/{templateId}/stages/{stageId}', [RecruitController::class, 'updateStage'])
            ->middleware(EnsurePermission::class.':recruiting.manage_templates');
        Route::delete('recruiting/stage-templates/{templateId}/stages/{stageId}', [RecruitController::class, 'destroyStage'])
            ->middleware(EnsurePermission::class.':recruiting.manage_templates');

        Route::get('job-openings', [RecruitController::class, 'listOpenings']);
        Route::post('job-openings', [RecruitController::class, 'storeOpening'])
            ->middleware(EnsurePermission::class.':recruiting.create');
        Route::get('job-openings/{jobOpeningId}', [RecruitController::class, 'showOpening']);
        Route::patch('job-openings/{jobOpeningId}', [RecruitController::class, 'updateOpening'])
            ->middleware(EnsurePermission::class.':recruiting.update');
        Route::delete('job-openings/{jobOpeningId}', [RecruitController::class, 'destroyOpening'])
            ->middleware(EnsurePermission::class.':recruiting.delete');
        Route::post('job-openings/{jobOpeningId}/toggle', [RecruitController::class, 'toggleOpening'])
            ->middleware(EnsurePermission::class.':recruiting.update');

        Route::post('job-openings/{jobOpeningId}/sponsors/{employeeId}', [RecruitController::class, 'addSponsor'])
            ->middleware(EnsurePermission::class.':recruiting.update');
        Route::delete('job-openings/{jobOpeningId}/sponsors/{employeeId}', [RecruitController::class, 'removeSponsor'])
            ->middleware(EnsurePermission::class.':recruiting.update');

        Route::get('job-openings/{jobOpeningId}/candidates', [RecruitController::class, 'listCandidates']);
        Route::get('job-openings/{jobOpeningId}/candidates/{candidateId}', [RecruitController::class, 'showCandidate']);

        Route::post('job-openings/{jobOpeningId}/candidates/{candidateId}/stages/{stageId}', [RecruitController::class, 'processStage']);

        Route::get('job-openings/{jobOpeningId}/candidates/{candidateId}/stages/{stageId}/notes', [RecruitController::class, 'listNotes']);
        Route::post('job-openings/{jobOpeningId}/candidates/{candidateId}/stages/{stageId}/notes', [RecruitController::class, 'storeNote']);
        Route::patch('job-openings/{jobOpeningId}/candidates/{candidateId}/stages/{stageId}/notes/{noteId}', [RecruitController::class, 'updateNote']);
        Route::delete('job-openings/{jobOpeningId}/candidates/{candidateId}/stages/{stageId}/notes/{noteId}', [RecruitController::class, 'destroyNote']);

        Route::post('job-openings/{jobOpeningId}/candidates/{candidateId}/stages/{stageId}/participants', [RecruitController::class, 'addParticipant']);
        Route::delete('job-openings/{jobOpeningId}/candidates/{candidateId}/stages/{stageId}/participants/{participantId}', [RecruitController::class, 'removeParticipant']);

        Route::post('job-openings/{jobOpeningId}/candidates/{candidateId}/hire', [RecruitController::class, 'hire'])
            ->middleware(EnsurePermission::class.':recruiting.hire');

        Route::get('job-openings/{jobOpeningId}/candidates/{candidateId}/files', [RecruitController::class, 'listFiles']);
        Route::post('job-openings/{jobOpeningId}/candidates/{candidateId}/files', [RecruitController::class, 'attachFile'])
            ->middleware(EnsurePermission::class.':recruiting.update');
        Route::delete('job-openings/{jobOpeningId}/candidates/{candidateId}/files/{mediaId}', [RecruitController::class, 'deleteFile'])
            ->middleware(EnsurePermission::class.':recruiting.update');
    });

Route::prefix('v1/jobs')->group(function () {
    Route::get('/', [PublicJobsController::class, 'listCompanies']);
    Route::get('{companySlug}', [PublicJobsController::class, 'listCompanyJobs']);
    Route::get('{companySlug}/jobs/{jobSlug}', [PublicJobsController::class, 'showJob']);
    Route::post('{companySlug}/jobs/{jobSlug}', [PublicJobsController::class, 'apply']);

    Route::get('{companySlug}/jobs/{jobSlug}/apply/{candidateUuid}/files', [PublicJobsController::class, 'listFiles']);
    Route::post('{companySlug}/jobs/{jobSlug}/apply/{candidateUuid}/files', [PublicJobsController::class, 'attachFile']);
    Route::delete('{companySlug}/jobs/{jobSlug}/apply/{candidateUuid}/files/{mediaId}', [PublicJobsController::class, 'deleteFile']);

    Route::post('{companySlug}/jobs/{jobSlug}/apply/{candidateUuid}/complete', [PublicJobsController::class, 'complete']);
    Route::delete('{companySlug}/jobs/{jobSlug}/apply/{candidateUuid}', [PublicJobsController::class, 'abandon']);
});
