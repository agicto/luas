# Knowledge Contract

The `knowledge` starter, installed by the `knowledge-base` plugin, owns organization knowledge
articles: a title and a Markdown body that owners and admins write and every member reads and
searches. It depends on the `user`, `audit`, and `organization` starters.

```env
OPTIONAL_STARTERS=organization,knowledge
```

## Routes

All routes require an authentication session and a verified `Organization-Id`.

| Method | Path | Who | Behavior |
|---|---|---|---|
| `GET` | `/v1/knowledge-articles` | Members | Keyset page of summaries, newest first (`cursor`, `per_page` 1–100) |
| `POST` | `/v1/knowledge-articles` | Owner, admin | Create; `201` with `ETag` |
| `GET` | `/v1/knowledge-articles/{id}` | Members | Read with `ETag` |
| `PATCH` | `/v1/knowledge-articles/{id}` | Owner, admin | Update `title` and/or `body`; requires `If-Match` |
| `DELETE` | `/v1/knowledge-articles/{id}` | Owner, admin | `204`; requires `If-Match` |
| `GET` | `/v1/knowledge-search?q=` | Members | Up to 20 ranked results with a 240-character excerpt |

`ETag` and `If-Match` carry `"knowledge-article-v<version>"`. A title is 1–200 characters after
trimming; a body is at most 200,000 characters of UTF-8 Markdown.

## Search

Search matches words with PostgreSQL full-text search (the language-neutral `simple`
configuration, title weighted above body) and also matches plain substrings of the title or body,
so languages written without spaces between words, such as Chinese, work. Results are ordered by
full-text rank and then by newest article. Substring matching scans the active organization's
articles; organizations with very large collections should add vector or trigram search.

## Errors

| HTTP | `error_code` | Meaning |
|---:|---|---|
| 400 | `COMMON.INVALID_INPUT` | Malformed cursor, ID, or `If-Match`, or a blank search query |
| 403 | `PERMISSION.DENIED` | A member tried to write |
| 404 | `KNOWLEDGE.ARTICLE_NOT_FOUND` | Not in the active organization (other organizations' IDs disclose nothing) |
| 412 | `KNOWLEDGE.VERSION_CONFLICT` | The article changed since it was read |
| 422 | `COMMON.VALIDATION_FAILED` | Field validation failed |
| 422 | `KNOWLEDGE.INVALID_ARTICLE` | Blank title or an update with no fields |
| 428 | `KNOWLEDGE.PRECONDITION_REQUIRED` | `If-Match` is missing |

## Privacy And Audit

Article bodies are never logged. Creates, updates, and deletes record an audit change with the
article ID and the changed field names; the title is recorded, the body only as `[changed]`.
Deleting an organization deletes its articles.
