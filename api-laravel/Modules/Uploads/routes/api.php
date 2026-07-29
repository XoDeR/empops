<?php

use Illuminate\Support\Facades\Route;
use Modules\Uploads\Http\Controllers\ChunkedUploadController;

Route::prefix('upload')->group(function () {
    Route::post('init', [ChunkedUploadController::class, 'init']);
    Route::post('chunk', [ChunkedUploadController::class, 'uploadChunk']);
    Route::post('complete', [ChunkedUploadController::class, 'completeUpload']);
    Route::get('status', [ChunkedUploadController::class, 'getUploadStatus']);

    // Optional endpoints for parity with file-uploads-go.
    Route::post('stream', [ChunkedUploadController::class, 'streamUpload']);
    Route::get('progress', [ChunkedUploadController::class, 'sseProgress']);
});

