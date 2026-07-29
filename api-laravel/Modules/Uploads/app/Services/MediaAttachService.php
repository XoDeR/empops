<?php

namespace Modules\Uploads\Services;

use Modules\Uploads\Models\TemporaryUpload;
use RuntimeException;
use Spatie\MediaLibrary\HasMedia;
use Spatie\MediaLibrary\MediaCollections\Models\Media;

final class MediaAttachService
{
    public function attachFromTemporary(
        HasMedia $target,
        string $collection,
        int $temporaryUploadId,
        int $mediaId,
    ): Media {
        $temp = TemporaryUpload::query()->find($temporaryUploadId);
        if ($temp === null) {
            throw new RuntimeException('Temporary upload not found', 404);
        }

        $media = $temp->getMedia('uploads')->firstWhere('id', $mediaId);
        if ($media === null) {
            throw new RuntimeException('Media not found on temporary upload', 404);
        }

        $target->clearMediaCollection($collection);

        return $media->move($target, $collection);
    }
}
