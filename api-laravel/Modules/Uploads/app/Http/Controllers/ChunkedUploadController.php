<?php

namespace Modules\Uploads\Http\Controllers;

use Carbon\Carbon;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Response;
use Modules\Uploads\Models\TemporaryUpload;

class ChunkedUploadController
{
    // Keep aligned with file-uploads-go defaults.
    private int $maxFileSizeBytes = 100 * 1024 * 1024;

    private function baseDir(): string
    {
        return storage_path('app/upload-chunks');
    }

    private function sessionDir(string $uploadId): string
    {
        return $this->baseDir() . DIRECTORY_SEPARATOR . $uploadId;
    }

    private function metaPath(string $uploadId): string
    {
        return $this->sessionDir($uploadId) . DIRECTORY_SEPARATOR . 'meta.json';
    }

    private function chunksDir(string $uploadId): string
    {
        return $this->sessionDir($uploadId) . DIRECTORY_SEPARATOR . 'chunks';
    }

    private function sanitizeFilename(string $filename): string
    {
        // Port of file-uploads-go validation.SanitizeFilename.
        $filename = str_replace('\\', '/', $filename);
        $slashPos = strrpos($filename, '/');
        if ($slashPos !== false) {
            $filename = substr($filename, $slashPos + 1);
        }

        $filename = str_replace(
            ['..', '/', "\0", '<', '>', ':', '"', '|', '?', '*'],
            ['', '_', '', '', '', '', '', '', '', '', ''],
            $filename,
        );

        if ($filename === '' || $filename === '.') {
            return 'unnamed_file';
        }

        return $filename;
    }

    private function totalChunks(int $totalSize, int $chunkSize): int
    {
        if ($totalSize <= 0 || $chunkSize <= 0) {
            throw new \InvalidArgumentException("invalid size parameters");
        }

        return intdiv($totalSize + $chunkSize - 1, $chunkSize);
    }

    private function expectedChunkSize(int $totalSize, int $chunkSize, int $chunkNumber, int $totalChunks): int
    {
        if ($chunkNumber === $totalChunks - 1) {
            return $totalSize - ($chunkNumber * $chunkSize);
        }

        return $chunkSize;
    }

    private function loadMeta(string $uploadId): array
    {
        $path = $this->metaPath($uploadId);
        if (!is_file($path)) {
            throw new \RuntimeException("Upload session not found", 404);
        }

        $raw = file_get_contents($path);
        if ($raw === false) {
            throw new \RuntimeException("Unable to read upload metadata", 500);
        }

        $meta = json_decode($raw, true);
        if (!is_array($meta)) {
            throw new \RuntimeException("Invalid upload metadata", 500);
        }

        return $meta;
    }

    private function missingChunks(array $meta): array
    {
        $uploadId = $meta['id'];
        $totalChunks = (int) $meta['total_chunks'];

        $missing = [];
        for ($i = 0; $i < $totalChunks; $i++) {
            $chunkPath = $this->chunksDir($uploadId) . DIRECTORY_SEPARATOR . 'chunk_' . $i;
            if (!is_file($chunkPath)) {
                $missing[] = $i;
            }
        }
        return $missing;
    }

    public function init(Request $request)
    {
        $payload = $request->json()->all();
        $filenameRaw = $payload['filename'] ?? null;
        $totalSizeRaw = $payload['total_size'] ?? null;
        $chunkSizeRaw = $payload['chunk_size'] ?? null;

        if (!is_string($filenameRaw) || $filenameRaw === '') {
            return Response::make('Invalid request body', 400);
        }

        if (!is_numeric($totalSizeRaw) || !is_numeric($chunkSizeRaw)) {
            return Response::make('Invalid size parameters', 400);
        }

        $totalSize = (int) $totalSizeRaw;
        $chunkSize = (int) $chunkSizeRaw;
        try {
            $totalChunks = $this->totalChunks($totalSize, $chunkSize);
        } catch (\Throwable $e) {
            return Response::make('Invalid size parameters', 400);
        }

        $uploadId = bin2hex(random_bytes(16));
        $filename = $this->sanitizeFilename($filenameRaw);

        $createdAt = Carbon::now();
        $expiresAt = Carbon::now()->addHours(24);

        $sessionDir = $this->sessionDir($uploadId);
        $chunksDir = $this->chunksDir($uploadId);
        @mkdir($chunksDir, 0755, true);

        $meta = [
            'id' => $uploadId,
            'filename' => $filename,
            'total_size' => $totalSize,
            'chunk_size' => $chunkSize,
            'total_chunks' => $totalChunks,
            'uploaded_chunks' => new \stdClass(), // empty object (parity with Go)
            'created_at' => $createdAt->toIso8601String(),
            'expires_at' => $expiresAt->toIso8601String(),
        ];

        file_put_contents($this->metaPath($uploadId), json_encode($meta));

        // Keep init payload compatible with upload-lib: { id, total_chunks, ... }.
        return response()->json($meta);
    }

    public function uploadChunk(Request $request)
    {
        $uploadId = (string) $request->query('upload_id', '');
        $chunkStr = (string) $request->query('chunk', '');

        if ($uploadId === '' || $chunkStr === '') {
            return Response::make('Invalid chunk number', 400);
        }

        if (!ctype_digit($chunkStr)) {
            return Response::make('Invalid chunk number', 400);
        }

        $chunkNumber = (int) $chunkStr;

        try {
            $meta = $this->loadMeta($uploadId);
        } catch (\Throwable $e) {
            return Response::make('Upload session not found', 404);
        }

        $totalChunks = (int) $meta['total_chunks'];
        if ($chunkNumber < 0 || $chunkNumber >= $totalChunks) {
            return Response::make('Invalid chunk number', 400);
        }

        $content = $request->getContent();
        $written = strlen($content);

        $expectedSize = $this->expectedChunkSize(
            (int) $meta['total_size'],
            (int) $meta['chunk_size'],
            $chunkNumber,
            $totalChunks
        );

        if ($written !== $expectedSize) {
            return Response::make('Chunk size mismatch', 400);
        }

        $chunkPath = $this->chunksDir($uploadId) . DIRECTORY_SEPARATOR . 'chunk_' . $chunkNumber;
        if (is_file($chunkPath)) {
            // Mirrors Go behaviour: chunk already uploaded.
            $uploadedCount = $totalChunks - count($this->missingChunks($meta));
            return response()->json([
                'chunk' => $chunkNumber,
                'uploaded' => $uploadedCount,
                'total' => $totalChunks,
            ], 200);
        }

        @mkdir($this->chunksDir($uploadId), 0755, true);
        $ok = file_put_contents($chunkPath, $content, LOCK_EX);
        if ($ok === false) {
            return Response::make('Error writing chunk', 500);
        }

        $missing = $this->missingChunks($meta);
        $uploadedCount = $totalChunks - count($missing);

        return response()->json([
            'chunk' => $chunkNumber,
            'uploaded' => $uploadedCount,
            'total' => $totalChunks,
        ], 200);
    }

    public function completeUpload(Request $request)
    {
        $uploadId = (string) $request->query('upload_id', '');
        if ($uploadId === '') {
            return Response::make('Upload session not found', 404);
        }

        try {
            $meta = $this->loadMeta($uploadId);
        } catch (\Throwable $e) {
            return Response::make('Upload session not found', 404);
        }

        $totalChunks = (int) $meta['total_chunks'];
        $missing = $this->missingChunks($meta);
        $uploadedCount = $totalChunks - count($missing);
        if ($uploadedCount !== $totalChunks) {
            return Response::make(
                sprintf('Missing chunks: %d/%d uploaded', $uploadedCount, $totalChunks),
                400
            );
        }

        $finalPath = $this->sessionDir($uploadId) . DIRECTORY_SEPARATOR . 'assembled_' . $meta['filename'];
        @mkdir(dirname($finalPath), 0755, true);

        $out = fopen($finalPath, 'wb');
        if ($out === false) {
            return Response::make('Error creating final file', 500);
        }

        for ($i = 0; $i < $totalChunks; $i++) {
            $chunkPath = $this->chunksDir($uploadId) . DIRECTORY_SEPARATOR . 'chunk_' . $i;
            $in = fopen($chunkPath, 'rb');
            if ($in === false) {
                fclose($out);
                return Response::make('Error reading chunk', 500);
            }

            while (!feof($in)) {
                $buf = fread($in, 1024 * 1024);
                if ($buf === false) {
                    fclose($in);
                    fclose($out);
                    return Response::make('Error assembling file', 500);
                }
                fwrite($out, $buf);
            }
            fclose($in);
        }
        fclose($out);

        // Store via Spatie Media Library into a temp model.
        $temp = TemporaryUpload::create();
        $media = $temp
            ->addMedia($finalPath)
            ->usingFileName($meta['filename'])
            ->toMediaCollection('uploads');

        // Best-effort cleanup of assembled file and chunk directory.
        @unlink($finalPath);
        $this->deleteSessionDir($uploadId);

        $mediaItem = $temp->getFirstMedia('uploads');

        return response()->json([
            'filename' => $meta['filename'],
            'size' => (int) $meta['total_size'],
            'status' => 'complete',
            'media_id' => $mediaItem?->id,
        ], 200);
    }

    private function deleteSessionDir(string $uploadId): void
    {
        $dir = $this->sessionDir($uploadId);
        if (!is_dir($dir)) {
            return;
        }

        $iterator = new \RecursiveIteratorIterator(
            new \RecursiveDirectoryIterator($dir, \FilesystemIterator::SKIP_DOTS),
            \RecursiveIteratorIterator::CHILD_FIRST
        );

        foreach ($iterator as $file) {
            /** @var \SplFileInfo $file */
            if ($file->isDir()) {
                @rmdir($file->getRealPath());
            } else {
                @unlink($file->getRealPath());
            }
        }

        @rmdir($dir);
    }

    public function getUploadStatus(Request $request)
    {
        $uploadId = (string) $request->query('upload_id', '');
        if ($uploadId === '') {
            return Response::make('Upload session not found', 404);
        }

        try {
            $meta = $this->loadMeta($uploadId);
        } catch (\Throwable $e) {
            return Response::make('Upload session not found', 404);
        }

        $totalChunks = (int) $meta['total_chunks'];
        $missing = $this->missingChunks($meta);
        $uploadedCount = $totalChunks - count($missing);

        return response()->json([
            'upload_id' => $meta['id'],
            'filename' => $meta['filename'],
            'uploaded_chunks' => $uploadedCount,
            'total_chunks' => $totalChunks,
            'missing_chunks' => $missing,
            'expires_at' => $meta['expires_at'],
        ], 200);
    }

    public function streamUpload(Request $request)
    {
        // upload-lib appends files using form.append("file", ...).
        $files = $request->file('file');
        if (!$files) {
            return Response::make('Invalid multipart request', 400);
        }

        $uploadId = (string) $request->header('X-Upload-ID', '');
        if ($uploadId === '') {
            $uploadId = bin2hex(random_bytes(16));
        }

        $temp = TemporaryUpload::create();

        $uploaded = [];
        $list = is_array($files) ? $files : [$files];
        foreach ($list as $file) {
            if (!$file) {
                continue;
            }
            $name = $this->sanitizeFilename($file->getClientOriginalName());
            $temp->addMedia($file->getRealPath())->usingFileName($name)->toMediaCollection('uploads');
            $uploaded[] = [
                'filename' => $name,
                'upload_id' => $uploadId,
            ];
        }

        return response()->json([
            'status' => 'success',
            'files' => $uploaded,
            'upload_id' => $uploadId,
            'message' => sprintf('Successfully uploaded %d file(s)', count($uploaded)),
        ], 200)->header('X-Upload-ID', $uploadId);
    }

    public function sseProgress(Request $request)
    {
        return Response::make('SSE progress is not implemented in this Laravel module yet.', 501);
    }
}

