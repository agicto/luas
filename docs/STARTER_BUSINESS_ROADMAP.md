# Starter Business Roadmap

This document records which starter-level business capabilities make Luas useful on day one for a downstream app, and which capabilities should be developed next.

Use [`../CONTEXT.md`](../CONTEXT.md) for vocabulary. A starter is a business-ready building block. A capability is a reusable technical integration that does not own an application workflow. Do not promote a capability into a starter until it owns a coherent workflow, contracts, tests, and replacement guidance.

## Current Ready-to-Use Baseline

| Surface | Current state | Ready for a new project? | Notes |
|---|---|---|---|
| `user` default starter | API registration, login, JWT auth, profile, password change, account deletion, password reset, auth abuse guard, seed user; Web mock auth plus same-origin production adapter | Yes | The adapter maps the browser contract to Go, keeps JWTs HttpOnly, preserves auth errors and rate-limit identity, and server-resolves protected sessions. Stateless logout cannot revoke an already issued JWT; see [`contracts/AUTHENTICATION.md`](../contracts/AUTHENTICATION.md). |
| `apikey` default starter | User-owned API key create/list/revoke, atomic hash-only persistence, exact scope guard, one-time plaintext, fixed production browser adapter, strict Web management UI, development mock | Yes | Good for developer tools, integrations, and AI/API products. Scopes attenuate the owner and do not replace RBAC. Usage metering is not included yet; see [`contracts/API_KEYS.md`](../contracts/API_KEYS.md). |
| `audit` default starter | Write-request audit middleware, route metadata, user-facing audit history, change metadata seam | Yes | Strong compliance baseline. It becomes more valuable once organization, permission, and resource ownership starters exist. |
| `organization` optional starter | Additive activation, organization/owner transaction, membership-scoped reads, request-scoped active context, owner/admin rename, invitation lifecycle, PII-minimized member directory, role/removal/leave policy, atomic ownership transfer, audit metadata, account-deletion membership guards; optional Web directory/create/URL switcher/context/profile/member/invitation/ownership workflow | Yes, when enabled | API, strict Web services, fixed production adapters, development mock state, role-aware UI, contracts, tests, and extraction guidance cover the reusable organization lifecycle. It remains opt-in and deliberately excludes organization deletion, durable email retries, and generalized RBAC. See [`contracts/ORGANIZATIONS.md`](../contracts/ORGANIZATIONS.md). |
| Web shell | Auth route group, protected console, settings page, devtools, mock BFF guardrails, i18n, typed env | Yes | Good scaffold workspace. It is intentionally replaceable and should not become a fixed downstream workspace. |
| Contracts | Global success/error envelopes, pagination, `error_code`, `request_id`, mock BFF expectations | Yes | Cross-starter endpoint contracts still need dedicated docs as new starters are added. |
| Capabilities | Crypto, ID generation, AI, workflow, events, email, storage, queue, schedule, tracing | Partly | Email now has typed all-or-none config, caller cancellation, a 10-second provider budget, bounded responses, and PII-safe errors, but delivery remains direct and best-effort. The memory workflow queue is bounded and race-free but process-local and non-durable; capabilities are not business-ready starters by themselves. |

## Architecture Review Findings

| Priority | Finding | Impact | Recommended slice |
|---|---|---|---|
| P1 | Permission/RBAC is documented as an optional starter decision, but no runnable `permission` starter is currently wired. | New teams may assume roles and permissions are available when only error vocabulary and examples remain. | Treat `permission` as a planned optional starter until its module, migrations, contracts, Web feature, and tests exist. |
| P1 | Notification capability exists, but no user-facing notification starter owns preferences, in-app records, or delivery status. | Apps repeatedly rebuild notification preferences and delivery history. | Build a `notification` optional starter backed by events, email, and optional in-app persistence. |
| P2 | Storage/R2 capability exists, but there is no file or asset starter with ownership, metadata, validation, signed URL, and deletion rules. | Upload features become ad hoc and security-sensitive. | Build a `file` or `asset` optional starter with storage abstraction and audit events. |
| P2 | App/workspace settings are represented by a console page, not by API-owned durable settings. | Downstream apps need feature flags, branding, locale, and workspace preferences. | Build a `setting` optional starter after organization ownership is clear. |
| P2 | API keys exist without usage metering, quota, billing, or plan limits. | Developer/API products need usage visibility and limits before production launch. | Build `usage` first, then keep `billing` optional and provider-adapted. |
| P2 | Event and workflow capabilities exist, but no webhook delivery starter owns subscriptions, signing, retry, and delivery logs. | Integration-heavy apps need outbound webhooks early. | Build a `webhook` optional starter using workflow retry primitives and audit logs. |
| P3 | AI capability exists, but no starter owns conversations, prompts, runs, evaluations, or cost tracking. | AI-first apps still need repeated product scaffolding. | Build an `ai-workspace` optional starter only after organization and usage seams are settled. |

## Recommended Starter Sequence

The production auth adapter is complete. The next business boundary is multi-user ownership; do
not add permission, billing, or workspace settings before deciding which organization owns them.

1. `organization` optional starter
   - Uses organization as the tenant/account boundary. Workspace is a possible future child concept, not a synonym in code or contracts.
   - The delivered starter owns organizations, membership-scoped reads, request-scoped active context, settings authorization, invitation onboarding, a privacy-minimized member directory, role/removal/leave flows, atomic ownership transfer, membership audit events, and account-deletion integrity guards.
   - Web owns directory, create, URL-scoped switching, context verification, rename, member administration, invitation management/acceptance, and ownership transfer through fixed production and mock adapters.
   - The starter remains optional; organization deletion, durable email retries, and generalized permission policy belong to later domain decisions.
   - Depends on `user`, `audit`, and email.
   - Web owns the organization switcher, member list, invitation flow, and organization settings.

2. `permission` optional starter
   - Owns roles, permissions, grants, policy checks, and route/service guard seams.
   - Depends on `organization` if permissions are workspace-scoped.
   - Should stay optional until the default scaffold proves that role complexity helps more apps than it slows down.

3. `notification` optional starter
   - Owns notification records, preferences, read state, delivery attempts, and user-facing notification center.
   - Uses events and email as capabilities.
   - Keeps SMS, Slack, or provider-specific channels as adapters, not starter vocabulary.

4. `file` or `asset` optional starter
   - Owns upload metadata, ownership, size/type validation, signed upload/download URLs, deletion policy, and audit events.
   - Uses storage/R2 capability.
   - Keeps direct storage SDK usage outside feature code.

5. `setting` optional starter
   - Owns typed settings at app, organization, and user scopes.
   - Useful for branding, locale defaults, notification preferences, and feature flags.
   - Must define which settings are public, private, cached, or audited.

6. `usage` optional starter before `billing`
   - Owns usage events, counters, quotas, and limit decisions.
   - Billing providers can be added later as adapters around a stable usage seam.
   - API keys, AI calls, storage, and workflows can all emit usage.

7. `webhook` optional starter
   - Owns endpoint subscriptions, signing secrets, delivery attempts, retry policy, and delivery logs.
   - Uses events and workflow primitives.
   - Gives integration-heavy downstream apps a production-grade outbound integration path.

8. `ai-workspace` optional starter
   - Owns prompt templates, conversation/session records, run history, cost attribution, and evaluation hooks.
   - Uses the AI capability, workflow, usage, and organization starters.
   - Should be optional because many Luas downstream apps will not be AI products.

## Starter Readiness Contract

A starter is not ready-to-use until it has all of these:

| Requirement | Evidence |
|---|---|
| Clear owner category | Default starter or optional starter decision documented in an ADR or this roadmap |
| Domain vocabulary | Framework-free entity/value/error seams in `api/internal/domain` only when shared |
| API implementation | `api/internal/modules/<starter>/` with service, repository, handler, routes, provider, tests |
| Starter manifest | Migrations, seeders, modules, middleware, and events registered through `internal/starter/assembly` |
| HTTP contract | Request, response, pagination, `error_code`, and `request_id` behavior documented under `contracts/` before Web integration |
| Web feature | `web/src/features/<feature>/` service, hooks, types, UI, tests, and route entry points when browser-facing |
| Mock BFF behavior | Development-only mock route handlers guarded by `guardMockBffRoute()` when Web needs backend-free local development |
| Audit behavior | Mutating operations emit audit metadata or explicitly document why audit does not apply |
| Downstream extraction | Keep/delete/replace guidance is clear in docs and does not leak downstream product assumptions |
| Verification | Targeted API/Web tests plus `make governance`; use `make check` when both halves move |

## Default vs Optional Decision Rule

Keep a starter optional when:

- The data model depends heavily on the downstream app domain.
- It changes permissions, tenancy, billing, storage policy, or user workflows.
- It adds visible console complexity for apps that may not need it.
- It depends on third-party providers that many apps will swap.

Promote a starter toward the default scaffold only when:

- Most downstream apps need it immediately.
- It has safe defaults and clear deletion paths.
- It improves security or auditability without forcing product-specific behavior.
- Its API and Web contracts are stable enough for repeated reuse.

## Near-Term Recommendation

Build `permission` next, then `notification`. The optional organization starter now gives those
starters a stable tenant term, role lifecycle, browser adapter, and persistence boundary without
pretending its three organization roles are generalized RBAC. File/asset, settings, usage, billing,
webhook, and AI workspace starters become much easier once permission scopes are settled.
