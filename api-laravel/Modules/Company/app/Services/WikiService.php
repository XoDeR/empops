<?php

namespace Modules\Company\Services;

use Illuminate\Support\Collection;
use Modules\Company\Models\Company;
use Modules\Company\Models\Wiki;
use Modules\Company\Models\WikiPage;
use Modules\Company\Models\WikiPageRevision;
use Modules\Employee\Models\Employee;

final class WikiService
{
    public function list(Company $company): Collection
    {
        return Wiki::query()->with('pages')->where('company_id', $company->id)->orderBy('title')->get();
    }

    public function create(Company $company, array $data): Wiki
    {
        return Wiki::query()->create(['company_id' => $company->id, ...$data]);
    }

    public function update(Wiki $wiki, array $data): Wiki { $wiki->fill($data)->save(); return $wiki->fresh('pages'); }
    public function delete(Wiki $wiki): void { $wiki->delete(); }

    public function createPage(Wiki $wiki, array $data, Employee $actor): WikiPage
    {
        $page = WikiPage::query()->create(['wiki_id' => $wiki->id, ...$data]);
        $this->revision($page, $actor);

        return $page;
    }

    public function showPage(WikiPage $page): WikiPage
    {
        $page->increment('pageviews_counter');

        return $page->fresh('revisions');
    }

    public function updatePage(WikiPage $page, array $data, Employee $actor): WikiPage
    {
        $page->fill($data)->save();
        $this->revision($page, $actor);

        return $page->fresh('revisions');
    }

    public function deletePage(WikiPage $page): void { $page->delete(); }

    private function revision(WikiPage $page, Employee $actor): void
    {
        WikiPageRevision::query()->create([
            'page_id' => $page->id,
            'employee_id' => $actor->id,
            'employee_name' => $actor->fullName(),
            'title' => $page->title,
            'content' => $page->content,
        ]);
    }
}
