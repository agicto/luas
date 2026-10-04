# Admin Console Starter Administration — Implementation Plan

Extends [admin-operator-console.md](admin-operator-console.md). Session transport, the `operator`
middleware, CSRF, Origin rules, and audit attribution are unchanged and are not repeated here.

## 1. Executive Summary

**Outcome.** A platform operator inspects tenants from the Admin Console: finds an organization,
sees its members, diagnoses its webhook endpoints and deliveries, replays a failed delivery, and
reviews the platform's notification delivery ledger.

**Problem.** The Admin Console manages users, audit, and app settings only. Organization, webhook,
and notification state is reachable only by tenant members through Web or by an operator with shell
access (`webhook:replay`, SQL). Support and incident work therefore needs database access.

**In scope.** Operator-authorized, read-mostly routes for the `organization`, `webhook`, and
`notification` starters under `/v1/operator`; matching Admin Console screens.

**Non-goals.** Moving tenant self-service UI out of Web (creating organizations, inviting, changing
roles, managing endpoints, reading a notification inbox stay customer workflows); editing or deleting
organizations or memberships as an operator; impersonation; disabling or editing webhook endpoints,
viewing signing secrets or event payloads; reading notification titles, bodies, or recipient
addresses; publishing notifications; retrying notification deliveries; permission, usage, and asset
administration.

**Deployable units.** `api/` and `admin/`, joined by `contracts/OPERATORS.md`.

**Success signal.** With `OPTIONAL_STARTERS=operator,organization,webhook,notification`, an operator
opens an organization in the Admin Console, sees its members and a failed webhook delivery, replays
it, and finds the replay in the audit log attributed to the operator. With a starter unselected, its
routes return `404` and its screens are absent.

**Decision from the owner (2026-10-02).** Bring the organization, notification, and webhook starters
into the Admin Console.

## 2. Actors and Access Surfaces

| Actor | Goal | Entry points | Read scope | Mutation scope | Denial behavior |
|---|---|---|---|---|---|
| Platform operator | Support and diagnose tenants | Admin Console via `/v1/operator/*` | All organizations and members (minimized), webhook endpoints, deliveries, attempts (secret-free), notification delivery metadata | Replay one terminal webhook delivery | `403 OPERATOR.FORBIDDEN` without a grant |
| Organization owner/admin | Manage own tenant | Web, public API | Own organization | Unchanged | Unchanged |
| Signed-in user without a grant | Use the product | Web | Own data | Own data | `403 OPERATOR.FORBIDDEN` before any lookup |

An operator is not an organization member and gains no organization role. Operator routes never set
organization context and never pass through `organization_context` middleware.

## 3. End-to-End Workflows

1. **Find a tenant.** Operator opens Organizations, searches by name or slug, opens one. Unknown ID:
   `404 ORGANIZATION.NOT_FOUND`.
2. **Inspect members.** The detail screen lists members with role and join time, paginated.
3. **Diagnose webhooks.** With `webhook` selected, the detail screen lists the organization's
   endpoints (status, disabled reason, consecutive failures) and deliveries filtered by status or
   endpoint; opening a delivery lists its attempts.
4. **Replay.** Operator confirms replay of a terminal delivery. The delivery returns to `pending`
   with the original message ID. Rejections: not terminal, replay limit reached, or endpoint not
   active → `409 WEBHOOK.REPLAY_NOT_ALLOWED`; unknown delivery → `404 WEBHOOK.DELIVERY_NOT_FOUND`.
5. **Review notification delivery.** With `notification` selected, the operator filters the
   delivery ledger by status, channel, or recipient user ID to see failures and their stable codes.
6. **Starter not selected.** Routes are not registered (`404 COMMON.NOT_FOUND`); the console build
   omits the screen unless the matching feature is listed in `VITE_OPTIONAL_FEATURES`.

## 4. Module Decomposition

| Module | Responsibility added | Data ownership | Consumers |
|---|---|---|---|
| `operator` | Mount starter-owned operator handlers behind `requireOperator` when the starter is selected | none added | Admin Console |
| `organization` | Platform-wide organization directory, organization lookup, member list without a membership requirement | `organizations`, `organization_memberships` | `operator` |
| `webhook` | Endpoint, delivery, and attempt reads and delivery replay addressed by an explicit organization ID | `webhook_*` | `operator` |
| `notification` | Delivery ledger read without content | `notification_deliveries`, `notifications` | `operator` |

The pattern is the one already used for app settings: the owning starter exposes handler methods
(`setting.Handler.AppList`), the `operator` starter owns authorization and mounting. No starter
reads another starter's tables. `operator` gains optional handler dependencies and no manifest
dependency; each group is mounted only when `Config.Starters.Selected(name)` is true.

- **Exclusions:** `operator` holds no SQL for these starters. Owning starters do not check operator
  grants; their operator handlers are unexported from routing and reachable only through `operator`.
- **Failure and recovery:** database outage → `503 COMMON.SERVICE_UNAVAILABLE`, retryable.

## 5. Domain Model and Vocabulary

No new terms. `Platform operator` in `CONTEXT.md` already states that an operator is not an
organization role. `Operator replay` is defined in `contracts/WEBHOOKS.md`; this plan adds an HTTP
entry point to the existing rule and changes none of it.

## 6. Persistence Design

No migration, no new column, no new index.

| Query | Access path |
|---|---|
| Organization directory, newest first | `organizations_pkey` descending; optional `ILIKE` on `name`/`slug` bounded by page size, same policy as the operator user search |
| Member counts for one page | `idx_organization_memberships_org_user` for at most 100 organization IDs |
| Members of one organization | `idx_organization_memberships_org_user` |
| Endpoints, deliveries, attempts of one organization | existing organization-leading webhook indexes; the queries are the tenant queries |
| Notification deliveries | `notification_deliveries_pkey` descending; `status`/`channel` filters use `idx_notification_deliveries_pending`; `user_id` joins `notifications` on its primary key |

Counts on the notification ledger are whole-table counts under the active filter, as for the operator
audit list. Retention keeps the table bounded; no cursor is introduced in this release.

## 7. HTTP Contract

All routes are under `/v1/operator`, require the operator session, send
`Cache-Control: private, no-store`, and use the shared envelope and `page`/`per_page` pagination
(default 15, max 100). Documented in `contracts/OPERATORS.md`.

| Method | Path | Operation | Requires starter | Success |
|---|---|---|---|---|
| `GET` | `/organizations` | Directory | `organization` | `200` page |
| `GET` | `/organizations/:id` | One organization | `organization` | `200` |
| `GET` | `/organizations/:id/members` | Members | `organization` | `200` page |
| `GET` | `/organizations/:id/webhook-endpoints` | Endpoints | `webhook` | `200` page |
| `GET` | `/organizations/:id/webhook-deliveries` | Deliveries | `webhook` | `200` page |
| `GET` | `/organizations/:id/webhook-deliveries/:delivery_id/attempts` | Attempts | `webhook` | `200` page |
| `POST` | `/organizations/:id/webhook-deliveries/:delivery_id/replay` | Replay | `webhook` + CSRF | `200` delivery |
| `GET` | `/notification-deliveries` | Delivery ledger | `notification` | `200` page |

### Error table

| Condition | HTTP | `error_code` | Retryable |
|---|---:|---|---|
| Malformed path, query, or filter | 400 | `COMMON.INVALID_INPUT` | No |
| No session / no grant / Origin / CSRF | 401/403 | as in the operator session contract | Per code |
| Organization does not exist | 404 | `ORGANIZATION.NOT_FOUND` | No |
| Delivery not in that organization | 404 | `WEBHOOK.DELIVERY_NOT_FOUND` | No |
| Delivery not terminal, replay limit reached, endpoint not active | 409 | `WEBHOOK.REPLAY_NOT_ALLOWED` | After state changes |
| Starter not selected | 404 | `COMMON.NOT_FOUND` | No |
| Database unavailable | 503 | `COMMON.SERVICE_UNAVAILABLE` | Yes |

Operator scope is global, so `404` discloses only that an ID does not exist.

### Queries

```text
QUERY ListOrganizations(operator, q, page, per_page)
VALIDATE q length 0..100
LOAD page FROM organization.organizations
  WHERE q empty OR name ILIKE %q% OR slug ILIKE %q%   (q escaped for LIKE wildcards)
  ORDER BY id DESC LIMIT per_page
LOAD member_count FROM organization.organization_memberships GROUP BY organization_id FOR page IDs
RETURN 200 { items: [{ id, name, slug, created_by, member_count, created_at, updated_at }], pagination }
```

```text
QUERY GetOrganization(operator, path.id)
LOAD organization USING path.id OR ELSE ORGANIZATION.NOT_FOUND
RETURN 200 { id, name, slug, created_by, member_count, created_at, updated_at }
```

```text
QUERY ListOrganizationMembers(operator, path.id, page, per_page)
LOAD organization USING path.id OR ELSE ORGANIZATION.NOT_FOUND
LOAD page FROM organization.organization_memberships JOIN undeleted users
  WHERE organization_id = path.id ORDER BY user_id ASC LIMIT per_page
RETURN 200 { items: [{ id, user_id, username, nickname, email, role, joined_at }], pagination }
```

Email is included because the operator user list already exposes it; the tenant member view stays
PII-minimized and unchanged.

```text
QUERY ListOrganizationWebhookEndpoints | ListOrganizationWebhookDeliveries | ListWebhookAttempts
LOAD organization USING path.id OR ELSE ORGANIZATION.NOT_FOUND
VALIDATE deliveries filter: endpoint_id positive integer; status IN delivery statuses
LOAD page FROM webhook store USING path.id   (existing tenant queries and response shapes)
RETURN 200 secret-free endpoint | delivery | attempt representations as in WEBHOOKS.md
```

```text
QUERY ListNotificationDeliveries(operator, status, channel, user_id, page, per_page)
VALIDATE status IN {pending, processing, delivered, failed}; channel IN {in_app, email}; user_id >= 1
LOAD page FROM notification.notification_deliveries JOIN notifications
  WHERE filters ORDER BY notification_deliveries.id DESC LIMIT per_page
RETURN 200 { items: [{ id, notification_id, user_id, kind, channel, status, attempts,
  last_failure_code, available_at, delivered_at, created_at, updated_at }], pagination }
```

Title, body, action URL, destination hash, and lease fields are never returned.

### POST /v1/operator/organizations/:id/webhook-deliveries/:delivery_id/replay

```text
COMMAND OperatorReplayWebhookDelivery(operator, path.id, path.delivery_id)
BIND operator FROM operator middleware (Origin and CSRF already enforced)
VALIDATE path.id, path.delivery_id positive integers ELSE COMMON.INVALID_INPUT
LOAD organization USING path.id OR ELSE ORGANIZATION.NOT_FOUND
BEGIN TRANSACTION webhook
  LOCK delivery WHERE id = path.delivery_id AND organization_id = path.id
    OR ELSE WEBHOOK.DELIVERY_NOT_FOUND
  REQUIRE delivery terminal AND replay_count < 100 AND endpoint active
    ELSE WEBHOOK.REPLAY_NOT_ALLOWED
  UPDATE delivery SET status = pending, cycle_attempt = 0, replay_count + 1, available_at = now
COMMIT
AUDIT webhook replay WITH actor = operator.id (existing webhook audit record)
RETURN 200 { delivery }
```

This is the existing `WebhookMaintainer.ReplayWebhookDelivery` rule, already reachable through
`webhook:replay`. Concurrency: the row lock makes concurrent replays serialize; the second sees a
non-terminal delivery and receives `409`. Not idempotent by design: each accepted replay is one new
delivery cycle, bounded by the replay limit.

## 8. State Machines

No new states. Webhook delivery `terminal → pending` on replay is the existing transition; the actor
set gains "platform operator through HTTP" beside "operator through CLI".

## 9. Authorization Matrix

| Capability | User | Organization owner/admin | Operator |
|---|---|---|---|
| Organization directory, any organization, any member list | forbidden | own only (Web) | all, read-only |
| Webhook endpoints, deliveries, attempts | forbidden | own (Web) | all, read-only, secret-free |
| Replay a webhook delivery | forbidden | not offered | allowed, audited |
| Create/update/disable/delete endpoints, rotate secrets | forbidden | own (Web) | forbidden |
| Notification inbox content | own | own | forbidden |
| Notification delivery metadata | forbidden | forbidden | all, read-only |

The API enforces every row. The console hides screens only.

## 10. Cross-Module Data Flow and Asynchronous Work

No new events, jobs, or schedules. Replay re-queues a delivery for the existing `webhook:work`
dispatcher. The global audit middleware records the unsafe request; the webhook starter records its
existing `webhook.replay` audit entry with the operator as actor.

## 11. Browser Surface (Admin Console)

| Route | Feature | Data | States |
|---|---|---|---|
| `/console/organizations` | `organizations` | directory with `q`, `page` in URL search | loading, empty, list, unavailable |
| `/console/organizations/$organizationId` | `organizations` + `webhooks` | organization, members; endpoints, deliveries (status filter), attempts dialog, replay confirm | loading, not found, unavailable, replay rejected |
| `/console/notifications` | `notification-deliveries` | ledger with `status`, `channel`, `userId`, `page` in URL search | loading, empty, list, unavailable |

- `VITE_OPTIONAL_FEATURES` gains `organization`, `webhook`, `notification`, mirroring the API
  starters. Each route requires `operator` and its own feature; navigation shows enabled features.
- TanStack Query owns server state; replay invalidates the organization's deliveries; no mutation
  retries. Zod validates every response. Copy in `en-US` and `zh-Hans`.
- Replay uses the existing confirm dialog pattern and shows the mapped error on `409`.

## 12. Security, Privacy, Audit, Operations

- **Data minimization:** no signing secret, ciphertext, payload, target response, notification
  content, or recipient address leaves the API on these routes. Endpoint URLs are shown because
  diagnosing delivery requires the target and organization managers already see them.
- **Tenant isolation:** every webhook read and the replay carry the path organization ID into the
  existing organization-scoped queries, so an ID from another organization is `404`.
- **Audit:** replay is recorded twice by existing mechanisms (request audit and webhook audit).
  Reads are not audited, consistent with the operator user list.
- **Disable switch:** remove the starter from `OPTIONAL_STARTERS`, or `operator` to remove all.

## 13. Test and Verification Strategy

| Concern | Proof |
|---|---|
| Directory search escaping, ordering, member counts, not-found | PostgreSQL test in `organization` |
| Member list without membership; deleted users excluded | PostgreSQL test in `organization` |
| Notification ledger filters, ordering, no content fields | PostgreSQL test in `notification` |
| Webhook operator handlers address the path organization; cross-organization delivery is `404` | Handler test with the store double |
| Routes mounted only when the starter is selected; operator guard precedes lookup | `operator` handler tests |
| Route catalog, governance boundaries | `make route-catalog-check`, `make governance` |
| Admin screens | Vitest component tests, type-check, lint, build |
| End to end | Local run against PostgreSQL: open organization, replay a failed delivery, see audit entry |

## 14. Delivery Plan

| Slice | Outcome | Units | Rollback |
|---|---|---|---|
| 1. Organization directory | Operator finds organizations and members | api, admin, contracts | Unselect `organization` in the console build |
| 2. Webhook diagnosis and replay | Operator sees endpoints, deliveries, attempts; replays | api, admin, contracts | Unselect `webhook` |
| 3. Notification delivery ledger | Operator filters deliveries | api, admin, contracts | Unselect `notification` |

Slice 2 depends on slice 1 (organization lookup and detail screen). Slice 3 is independent.

## 15. Acceptance Criteria

1. Given an operator who is a member of no organization, when they list organizations, then all
   organizations are returned newest first with member counts.
2. Given an unknown organization ID, any nested operator route returns `404 ORGANIZATION.NOT_FOUND`.
3. Given a delivery of organization A, when addressed under organization B, then attempts and replay
   return `404 WEBHOOK.DELIVERY_NOT_FOUND` and nothing changes.
4. Given a failed delivery on an active endpoint, when the operator replays it, then it is `pending`
   with `replay_count + 1` and the audit log names the operator.
5. Given a pending delivery, replay returns `409 WEBHOOK.REPLAY_NOT_ALLOWED`.
6. Given notification deliveries, the ledger response contains no title, body, or action URL.
7. Given `webhook` is not selected, its operator routes are absent and the organization screen shows
   no webhook section.
8. A signed-in user without a grant receives `403 OPERATOR.FORBIDDEN` on every route above.

## 15a. Delivery Status (2026-10-02)

All three slices are delivered and merged to `main`, verified with PostgreSQL
tests for each new query, handler tests for mounting and guard order, Admin component tests, and a
live browser run against a real API and database:

| Slice | Evidence |
|---|---|
| 1. Organization directory | An operator who is a member of nothing lists and opens organizations; members show email; unknown ID is `404 ORGANIZATION.NOT_FOUND` |
| 2. Webhook diagnosis and replay | Endpoints and deliveries render without secrets; a delivery addressed under another organization is `404`; replay of a pending delivery is `409`; replay of a delivered one returns it to `pending` with `replay_count` 1 and appears in the audit log with the operator as actor |
| 3. Notification ledger | Filters by status, channel, and recipient; the response carries no title, body, or action URL |

Refinement recorded during delivery: the default page size is the repository default of 15, and the
console requests 20 (50 for the notification ledger and attempts).

## 16. Open Decisions

None blocking. Recorded assumptions:

- **A1.** "Bring starters into Admin" means operator support views, not a second tenant UI. Tenant
  self-service stays in Web because the Admin Console authenticates operators, not members.
- **A2.** The only mutation is webhook replay, because it is already an operator capability. Operator
  disable of an endpoint is deferred: a tenant could re-enable it, so a real kill switch needs a new
  disabled reason and rule.
- **A3.** Notification content is never shown to operators. Retrying a failed notification delivery
  is deferred: it adds a `failed → pending` transition that does not exist today.
- **A4.** Webhook data is reached through an organization, never as a cross-tenant list, so existing
  organization-leading indexes serve every query.

## 17. Readiness Gate

- Product and domain: outcome, non-goals, actors, and rejection flows are defined; no new vocabulary
  or state.
- Ownership: each starter keeps sole ownership of its tables and rules; `operator` owns
  authorization and mounting only.
- Data and contract: no migration; routes, filters, bounds, ordering, error codes, and replay
  concurrency are defined.
- Side effects: none new; replay reuses the existing dispatcher and audit records.
- Delivery: three slices with acceptance criteria and PostgreSQL proof where SQL behavior matters.

readiness: READY WITH RECORDED ASSUMPTIONS
