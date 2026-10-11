# Comments Contract

The `comments` starter, installed by the `comments` plugin, owns threaded comments on organization
resources. It depends on the `user`, `audit`, and `organization` starters, and uses the
`notification` starter for @mention notifications only when that starter is also selected.

```env
OPTIONAL_STARTERS=organization,comments          # mentions stored, not notified
OPTIONAL_STARTERS=organization,notification,comments
```

## Resource Types

A comment belongs to one resource, named by `resource_type` and `resource_id`. Resource types are a
code-owned catalog: the plugin ships none, and each module that wants comments registers its type
in `ProvideResourceCatalog` (`api/internal/modules/comments/resources.go`):

```go
func ProvideResourceCatalog(tasks domain.TaskReader) (*ResourceCatalog, error) {
	return NewResourceCatalog(ResourceType{
		Name: "task",
		Checker: ResourceCheckerFunc(func(ctx context.Context, org domain.OrganizationContext, id string) (ResourceAccess, error) {
			visible, err := tasks.VisibleTo(ctx, org.OrganizationID, org.UserID, id)
			return ResourceAccess{Read: visible, Comment: visible}, err
		}),
		ActionURL: func(id string, commentID uint) string { return "/console/tasks/" + id },
	})
}
```

The checker runs on every request with the verified organization context. It answers whether the
resource exists in that organization and whether the caller may read its comments (`Read`) and
write them (`Comment`). A resource that does not exist, belongs to another organization, or is not
visible to the caller answers `404`, so identifiers disclose nothing. `ActionURL` is the optional
same-origin link of a mention notification.

| Field | Rule |
|---|---|
| `resource_type` | Registered name: lowercase dotted words, at most 40 characters |
| `resource_id` | 1–64 characters: letters, digits, `.`, `_`, `:`, `-`; starts with a letter or digit |

## Routes

All routes require an authentication session and a verified `Organization-Id`.

| Method | Path | Who | Behavior |
|---|---|---|---|
| `GET` | `/v1/comments?resource_type=&resource_id=` | `Read` | Keyset page of threads, oldest first (`cursor`, `per_page` 1–100); each root with its first 3 replies and `has_more_replies` |
| `GET` | `/v1/comments/{id}/replies` | `Read` | Keyset page of one thread's replies, oldest first |
| `POST` | `/v1/comments` | `Comment` | Create a comment, or a reply with `parent_id`; `201` with `ETag` |
| `PATCH` | `/v1/comments/{id}` | Author with `Comment` | Edit `body` and/or `mentions`; requires `If-Match` |
| `DELETE` | `/v1/comments/{id}` | Author, owner, admin (with `Read`) | Soft delete; `204` |

`ETag` and `If-Match` carry `"comment-v<version>"`. Every write raises `version`.

## Comments And Threads

- A body is 1–10,000 characters of plain text or Markdown after trimming surrounding whitespace.
  Clients render it as text or sanitized Markdown, never as raw HTML.
- Threads are one level deep. A reply's `parent_id` is always a root comment; replying to a reply
  attaches the new comment to that reply's root. The parent must be a live comment on the same
  resource, otherwise `422 COMMENTS.INVALID_PARENT`.
- An edit sets `edited_at`. Only the author edits; owners and admins moderate by deleting.
- A delete is soft: `body` becomes `null`, `mentions` becomes empty, and `deleted_at` is set, so
  replies keep their thread. The text is erased from the database, not hidden. A deleted comment
  cannot be edited, deleted again, or replied to (`404` / `422`).
- `author_id` becomes `null` when the author's account is deleted. Deleting the organization
  deletes its comments; deleting a root deletes its replies.

A comment:

```json
{
  "id": 41,
  "resource_type": "task",
  "resource_id": "1042",
  "parent_id": null,
  "author_id": 7,
  "body": "Can @Dana review this?",
  "mentions": [9],
  "version": 1,
  "created_at": "2026-10-11T09:00:00Z",
  "updated_at": "2026-10-11T09:00:00Z",
  "edited_at": null,
  "deleted_at": null
}
```

A list item is `{ "comment": Comment, "replies": [Comment], "has_more_replies": boolean }`.

## Mentions

`mentions` is a list of at most 20 distinct user IDs, each a member of the active organization;
anything else is `422 COMMENTS.INVALID_MENTIONS`. The body text is not parsed: the client sends the
IDs of the members it rendered as mentions. Editing `mentions` replaces the list.

When the `notification` starter is selected, every newly mentioned member except the author gets one
notification of kind `comments.mention` on the `in_app` and `email` channels (subject to the
member's preferences) with idempotency key `comments.mention:<comment_id>:<user_id>`. Retries and
re-mentions therefore never notify twice. The notification names the resource, never the comment
text. Publication happens after the comment is stored; a failure is logged with identifiers only
and does not fail the request. Without the `notification` starter, mentions are stored and nobody is
notified.

## Errors

| HTTP | `error_code` | Meaning |
|---:|---|---|
| 400 | `COMMON.INVALID_INPUT` | Malformed cursor, ID, `resource_id`, or `If-Match` |
| 403 | `PERMISSION.DENIED` | No `Comment` access, editing someone else's comment, or deleting it without being owner or admin |
| 404 | `COMMENTS.RESOURCE_NOT_FOUND` | The resource is not in the organization or not visible to the caller |
| 404 | `COMMENTS.COMMENT_NOT_FOUND` | Not in the organization, on an invisible resource, or (for writes) deleted |
| 412 | `COMMENTS.VERSION_CONFLICT` | The comment changed since it was read |
| 422 | `COMMON.VALIDATION_FAILED` | Field validation failed |
| 422 | `COMMENTS.UNSUPPORTED_RESOURCE_TYPE` | `resource_type` is not registered |
| 422 | `COMMENTS.INVALID_COMMENT` | Blank body or an edit with no fields |
| 422 | `COMMENTS.INVALID_PARENT` | Parent missing, deleted, or on another resource |
| 422 | `COMMENTS.INVALID_MENTIONS` | More than 20, duplicates, or non-members |
| 428 | `COMMENTS.PRECONDITION_REQUIRED` | `If-Match` is missing |

## Privacy And Audit

Comment bodies are never logged, audited, or copied into notifications. Creates, edits, and deletes
record an audit change with the comment ID, the resource, and the changed field names; an edited
body is recorded only as `[changed]` and mentions only as a count. A delete by an owner or admin of
someone else's comment is marked `moderated` with the author's ID.

## Bounds

Pages hold at most 100 threads with at most 3 replies each, so one list request reads at most
400 comments. Each request runs one resource check and at most 20 membership checks. Apply the
deployment's normal rate limits to `POST /v1/comments`.
