# Setting Contract

The optional `setting` starter owns durable, typed overrides for a code-owned setting catalog.
It depends on the default `user` and `audit` starters and on the optional `organization` starter.
The matching Web feature therefore requires both `organization` and `setting`.

A **setting definition** is code-owned metadata: key, scope, kind, visibility, default, and
validation policy. A **setting override** is one durable value selected for an app, organization,
or user. The effective **setting** combines the definition with either its override or its default.
Definitions are never created through HTTP or stored as arbitrary database records.

Settings are not process configuration, secrets, permissions, entitlements, usage limits, or
notification channel preferences. Secrets remain in environment/provider configuration;
permissions remain with `permission`; notification preferences remain with `notification`.

## Activation

Enable the API starter for API, migration, and operator-command processes:

```dotenv
OPTIONAL_STARTERS=organization,setting
```

Enable the matching browser features in the same Web build:

```dotenv
NEXT_PUBLIC_OPTIONAL_FEATURES=organization,setting
```

Selection is additive and restart-scoped. Enabling only `setting` is invalid because organization
scope and its verified membership context are part of this starter's stable contract.

## Code-Owned Catalog

The shipped catalog is intentionally small:

| Scope | Key | Kind | Visibility | Default |
|---|---|---|---|---|
| app | `branding.display_name` | string | public | `Luas` |
| app | `localization.locale` | enum | public | `en-US` |
| organization | `localization.locale` | enum | private | `en-US` |
| user | `localization.locale` | enum | private | `en-US` |
| user | `localization.timezone` | timezone | private | `UTC` |

Locale values are `en-US` or `zh-Hans`. Time zones must be valid IANA names. Downstream apps may
compose more string, boolean, integer, enum, or timezone definitions at assembly time. Keys use
lowercase namespaced dot-separated semantic names (`domain.reason`, with letter-led snake_case
segments). The catalog is capped at 64 definitions and each encoded
value at 4096 bytes; arbitrary objects and arrays are not supported.

Removing or changing a definition is a code migration decision. Rows for unknown keys are never
returned and do not turn into an unbounded dynamic configuration surface.

## Effective Values And Versions

An effective setting has this shape:

```json
{
  "scope": "user",
  "key": "localization.timezone",
  "kind": "timezone",
  "visibility": "private",
  "value": "Europe/Dublin",
  "version": 3,
  "source": "override",
  "updated_at": "2026-07-15T21:00:00Z"
}
```

Enum definitions also expose their fixed `options`. A definition with no durable history has
version `0`, source `default`, and `updated_at: null`. Setting an override increments the version.
Resetting retains a tombstone, increments the version, and exposes the code default with source
`default`; this prevents an old writer from becoming current again after reset. A no-op write does
not increment the version.

Every mutation requires exactly one strong item precondition:

```http
If-Match: "setting-v3"
```

Missing preconditions return `428 SETTING.PRECONDITION_REQUIRED`. Malformed, wildcard, weak, or
multiple validators return `400 COMMON.INVALID_INPUT`. A stale version returns
`412 SETTING.VERSION_CONFLICT`. A client must refetch instead of silently overwriting another
writer.

## HTTP API

Collections are not paginated because the code-owned catalog is hard-capped at 64 definitions.

| Operation | Endpoint | Authorization | Success |
|---|---|---|---|
| Public app settings | `GET /v1/settings/public` | Public definitions only | Effective app settings |
| Current-user settings | `GET /v1/settings/user` | Bearer user | Effective user settings |
| Set current-user override | `PATCH /v1/settings/user/:key` | Bearer user + `If-Match` | Effective setting |
| Reset current-user override | `DELETE /v1/settings/user/:key` | Bearer user + `If-Match` | HTTP 204 |
| Active-organization settings | `GET /v1/organization-settings` | Verified organization member | Effective organization settings |
| Set organization override | `PATCH /v1/organization-settings/:key` | Owner/admin + `If-Match` | Effective setting |
| Reset organization override | `DELETE /v1/organization-settings/:key` | Owner/admin + `If-Match` | HTTP 204 |

`PATCH` accepts exactly one field and one scalar value:

```json
{
  "value": "zh-Hans"
}
```

Unknown fields, trailing JSON, objects, arrays, `null`, values outside the definition schema, and
unknown or wrong-scope keys fail closed. Unknown keys return `404 SETTING.NOT_FOUND`; invalid values
return `422 SETTING.INVALID_VALUE`. Organization reads require a verified `Organization-Id`
context; only owner/admin membership roles may mutate. The optional permission starter does not
silently replace that lifecycle rule.

Successful item mutations emit an `ETag` matching the returned version. Successful reset emits the
new item `ETag` even though the body is empty.

## Cache Contract

`GET /v1/settings/public` returns only public app definitions with:

```http
Cache-Control: public, max-age=60, stale-while-revalidate=300
ETag: "settings-<sha256>"
```

`If-None-Match` accepts that aggregate validator and returns HTTP 304 with no body when unchanged.
Every authenticated setting response uses `Cache-Control: private, no-store`; user responses vary
on `Cookie`, and organization responses vary on both `Cookie` and `Organization-Id` at the browser
adapter boundary. Public organization settings are deliberately absent because Luas has no public
organization identity or enumeration contract.

## Persistence And Deletion

One row is unique by `(scope, subject_id, key)`. Scope ownership is explicit: app uses subject `0`,
user rows reference one user, and organization rows reference one organization. Writes lock the
owning user or organization row before compare-and-swap so account/tenant lifecycle races cannot
attach overrides to a disappearing owner.

Account deletion removes user-scope setting rows inside the same transaction after all deletion
guards pass and before the user is soft-deleted. A failed guard runs no cleanup; a failed cleanup
rolls back deletion. Organization setting rows are retained until a future organization-deletion
contract exists and are protected by the database ownership constraint.

## Audit And Privacy

Mutations record only scope, key, subject identity, prior/new version, source, and operation. Values,
defaults, enum options, request bodies, and stored JSON never enter audit changes, logs, metrics, or
error messages. Settings are not a secrets facility even with this minimization.

The operator commands `setting:list`, `setting:set`, and `setting:reset` manage app-scope overrides.
Set/reset require an explicit expected version; command output contains key/version/source metadata,
not values. Set/reset also write a minimized system-actor audit entry. As with request audit
persistence, an audit-store failure is reported as an operator warning after the setting commit and
does not make a completed mutation appear rolled back.

## Stable Errors

| HTTP status | `error_code` | Meaning |
|---|---|---|
| 404 | `SETTING.NOT_FOUND` | The key is unknown or belongs to another scope |
| 412 | `SETTING.VERSION_CONFLICT` | `If-Match` does not match the durable version |
| 422 | `SETTING.INVALID_VALUE` | The scalar value violates the code-owned schema |
| 428 | `SETTING.PRECONDITION_REQUIRED` | A mutation omitted `If-Match` |
| 503 | `COMMON.SERVICE_UNAVAILABLE` | Persistence or stored-value decoding is unavailable |

Global envelopes, authentication errors, validation behavior, and `request_id` follow
[`README.md`](README.md).

## Deliberate Deferrals

The starter does not ship runtime definition creation, arbitrary JSON, secrets, dynamic remote
feature-flag evaluation, percentage rollout, entitlement policy, public organization branding,
notification preferences, bulk mutation, or hot-reloaded process configuration. Those concerns need
their own ownership, consistency, and security contracts rather than expanding `setting` by accident.
