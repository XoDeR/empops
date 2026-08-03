<?php

namespace Modules\Company\Models;

use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class WikiPage extends Model
{
    use HasUuids;
    protected $fillable = ['wiki_id', 'title', 'content', 'pageviews_counter'];
    protected function casts(): array { return ['pageviews_counter' => 'integer']; }
    public function wiki(): BelongsTo { return $this->belongsTo(Wiki::class); }
    public function revisions(): HasMany { return $this->hasMany(WikiPageRevision::class, 'page_id')->latest(); }
}
