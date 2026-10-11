# Tags Contract

The `tag` starter, installed by the `tags` plugin, owns organization tags: short labels with a
palette color that resources of code-registered types can carry. Owners and admins manage tags and
assignments; every member reads them. It depends on the `user`, `audit`, and `organization`
starters.

```env
OPTIONAL_STARTERS=organization,tag
```

## Routes

All routes require an authentication session and a verified `Organization-Id`.

| Method | Path | Who | Behavior |
|---|---|---|---|
| `GET` | `/v1/tags` | Members | Keyset page ordered by lowercase name (`cursor`, `per_page` 1–100) |
| `POST` | `/v1/tags` | Owner, admin | Create; `201` with `ETag` |
| `GET` | `/v1/tags/{id}` | Members | Read with `ETag` |
| `PATCH` | `/v1/tags/{id}` | Owner, admin | Update `name`, `color`, and/or `description`; requires `If-Match` |
| `DELETE` | `/v1/tags/{id}` | Owner, admin | `204`; removes the tag from every resource; requires `If-Match` |
| `GET` | `/v1/tags/{id}/resources` | Members | Keyset page of resources carrying the tag, newest assignment first; optional `resource_type` filter |
| `GET` | `/v1/resource-tags/{resource_type}/{resource_id}` | Members | The resource's tags (at most 20), ordered by name |
| `PUT` | `/v1/resource-tags/{resource_type}/{resource_id}` | Owner, admin | Replace the resource's complete tag set with `{"tag_ids": [...]}` |

`ETag` and `If-Match` carry `"tag-v<version>"`.

## Tags

- `name` is 1–50 characters of single-line text after trimming, unique in the organization
  ignoring case. Renaming a tag to a different case of its own name is allowed.
- `color` is one of the palette token names `gray`, `red`, `orange`, `yellow`, `green`, `blue`,
  `purple`, `pink`; it defaults to `gray`. Browsers map the names to their theme tokens.
- `description` is at most 200 characters and defaults to empty.
- Lists order by lowercase name in byte order, so the order is the same on every database locale.

## Resource Types And Assignments

A resource type names what a tag is attached to, for example `knowledge_article`. Types are a
code-owned catalog: the module that owns a resource registers its type and a check that the
resource exists in the organization, at application assembly time. The starter ships with no
types; until a module registers one, every assignment route answers `TAG.RESOURCE_NOT_FOUND`.

- `resource_type` matches `^[a-z][a-z0-9_]{0,49}$`; `resource_id` matches
  `^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`.
- Reading or replacing a resource's tags first asks the owning module whether the resource is in
  the active organization; an unregistered type and another organization's resource both answer
  `404 TAG.RESOURCE_NOT_FOUND`.
- `PUT` makes `tag_ids` the complete set: at most 20 unique IDs of the active organization's tags;
  `[]` removes every tag. Repeating a request changes nothing, and concurrent replacements of one
  resource end with exactly one request's set.
- Deleting a tag removes its assignments. Owning modules remove a deleted resource's assignments
  through the starter's `DetachResource` service method; the resources list may otherwise show a
  resource that its module has since deleted.

## Errors

| HTTP | `error_code` | Meaning |
|---:|---|---|
| 400 | `COMMON.INVALID_INPUT` | Malformed cursor, ID, `resource_type`, `resource_id`, or `If-Match` |
| 403 | `PERMISSION.DENIED` | A member tried to write |
| 404 | `TAG.NOT_FOUND` | The tag is not in the active organization (other organizations' IDs disclose nothing) |
| 404 | `TAG.RESOURCE_NOT_FOUND` | The resource type is not registered or the resource is not in the active organization |
| 409 | `TAG.NAME_TAKEN` | Another tag of the organization has the name, ignoring case |
| 412 | `TAG.VERSION_CONFLICT` | The tag changed since it was read |
| 422 | `COMMON.VALIDATION_FAILED` | Field validation failed, including a missing `tag_ids` or more than 20 |
| 422 | `TAG.INVALID` | Blank name, unknown color, or an update with no fields |
| 422 | `TAG.INVALID_ASSIGNMENT` | A repeated tag, more than 20 tags, or a tag that is not in the organization |
| 428 | `TAG.PRECONDITION_REQUIRED` | `If-Match` is missing |

## Privacy And Audit

Tag creates, updates, and deletes record an audit change with the tag ID, its name, and its color;
the description only as `[changed]`. A changed assignment set records the resource type, the
resource ID, and the tag IDs before and after; an unchanged set records nothing. Responses are
private and never cached. Deleting an organization deletes its tags and assignments.
