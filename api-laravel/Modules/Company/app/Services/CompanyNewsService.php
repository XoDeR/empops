<?php

namespace Modules\Company\Services;

use Modules\Company\Models\Company;
use Modules\Company\Models\CompanyNews;
use Modules\Employee\Models\Employee;

final class CompanyNewsService
{
    public function __construct(private readonly AuditLogger $audit) {}

    /**
     * @param  array{title: string, content: string}  $data
     */
    public function create(Company $company, Employee $actor, array $data): CompanyNews
    {
        $news = CompanyNews::query()->create([
            'company_id' => $company->id,
            'author_id' => $actor->id,
            'author_name' => $actor->fullName(),
            'title' => trim($data['title']),
            'content' => trim($data['content']),
        ]);

        $this->audit->log($company, $actor, 'company_news.created', $news, [
            'title' => $news->title,
        ]);

        return $news;
    }

    /**
     * @param  array{title?: string, content?: string}  $data
     */
    public function update(CompanyNews $news, array $data, Employee $actor): CompanyNews
    {
        if (isset($data['title'])) {
            $news->title = trim($data['title']);
        }
        if (isset($data['content'])) {
            $news->content = trim($data['content']);
        }
        $news->save();

        $this->audit->log($news->company, $actor, 'company_news.updated', $news, [
            'title' => $news->title,
        ]);

        return $news->fresh();
    }

    public function destroy(CompanyNews $news, Employee $actor): void
    {
        $company = $news->company;
        $payload = ['title' => $news->title, 'news_id' => (string) $news->id];
        $news->delete();
        $this->audit->log($company, $actor, 'company_news.deleted', null, $payload);
    }

    /**
     * @return array<string, mixed>
     */
    public function payload(CompanyNews $news): array
    {
        return [
            'id' => (string) $news->id,
            'company_id' => (string) $news->company_id,
            'author_id' => $news->author_id ? (string) $news->author_id : null,
            'author_name' => $news->author_name,
            'title' => $news->title,
            'content' => $news->content,
            'created_at' => $news->created_at?->toIso8601String(),
            'updated_at' => $news->updated_at?->toIso8601String(),
        ];
    }
}
