<?php

namespace Modules\Uploads\Models;

use Illuminate\Database\Eloquent\Model;
use Spatie\MediaLibrary\InteractsWithMedia;
use Spatie\MediaLibrary\HasMedia;

class TemporaryUpload extends Model implements HasMedia
{
    use InteractsWithMedia;

    protected $fillable = [];

    public function registerMediaCollections(): void
    {
        // Upload-lib sends either one assembled file per chunked session,
        // or multiple files per "stream" request.
        $this->addMediaCollection('uploads');
    }
}

