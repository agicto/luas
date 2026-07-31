# Starter Business Roadmap

This document records which starter-level business capabilities make Luas useful on day one for a downstream app, and which capabilities should be developed next.

Use [`../CONTEXT.md`](../CONTEXT.md) for vocabulary. A starter is a business-ready building block. A capability is a reusable technical integration that does not own an application workflow. Do not promote a capability into a starter until it owns a coherent workflow, contracts, tests, and replacement guidance.

## Current Ready-to-Use Baseline

| Surface | Current state | Ready for a new project? | Notes |
|---|---|---|---|
| `user` default starter | API registration, login, hash-only opaque authentication sessions, profile, password change, account deletion, password reset, auth abuse guard, session retention CLI, seed user; Web mock auth plus same-origin production adapter | Yes | The adapter keeps the credential HttpOnly, preserves auth errors and rate-limit identity, and performs remote idempotent logout. Current persistence owns revocation, idle/absolute expiry, account status, and password-event invalidation; see [`contracts/AUTHENTICATION.md`](../contracts/AUTHENTICATION.md). |
| `apikey` default starter | User-owned API key create/list/revoke, atomic hash-only persistence, exact scope guard, one-time plaintext, fixed production browser adapter, strict Web management UI, development mock | Yes | Good for developer tools, integrations, and AI/API products. Scopes attenuate the owner and do not replace RBAC. The optional usage starter supplies owner-level metering and quota seams; API-key attribution remains an explicit producer decision. See [`contracts/API_KEYS.md`](../contracts/API_KEYS.md). |
| `audit` default starter | Write-request audit middleware, route metadata, user-facing audit history, change metadata seam, explicit bounded retention command | Yes | Strong investigation baseline with a reviewed HTTP contract and PostgreSQL retention index. It does not claim legal hold, WORM storage, or a universal retention period. See [`contracts/AUDIT.md`](../contracts/AUDIT.md). |
| `organization` optional starter | Additive activation, organization/owner transaction, membership-scoped reads, request-scoped active context, owner/admin rename, invitation lifecycle, PII-minimized member directory, role/removal/leave policy, atomic ownership transfer, audit metadata, account-deletion membership guards; optional Web directory/create/URL switcher/context/profile/member/invitation/ownership workflow | Yes, when enabled | API, strict Web services, fixed production adapters, development mock state, role-aware UI, contracts, tests, and extraction guidance cover the reusable organization lifecycle. It remains opt-in and deliberately excludes organization deletion, durable email retries, and generalized RBAC. See [`contracts/ORGANIZATIONS.md`](../contracts/ORGANIZATIONS.md). |
| `permission` optional starter | Organization-scoped access roles, code-owned exact permission catalog, current-persistence authorizer, owner bypass, delegated-management dominance checks, transactional assignment replacement, route guard, audit metadata; optional strict Web role/member management and mock parity | Yes, when enabled with `organization` | It is allow-only and default-deny, with no direct user grants, wildcards, role hierarchy, explicit deny, or resource-instance policy language. Product modules extend the catalog at assembly time and keep ownership checks local. See [`contracts/PERMISSIONS.md`](../contracts/PERMISSIONS.md). |
| `notification` optional starter | Idempotent internal publication, user preferences, in-app records/read state, durable email delivery ledger, lease worker, stable failure codes; optional strict Web notification center and mock parity | Yes, when enabled | It is user-scoped and independent of organization. Required channels can override future-delivery preferences, email retries use stable provider idempotency, and no public publish endpoint or recipient/provider detail enters the browser contract. See [`contracts/NOTIFICATIONS.md`](../contracts/NOTIFICATIONS.md). |
| `asset` optional starter | User-owned private metadata, idempotent upload intents, staging-to-final promotion, bounded content inspection, short-lived transfer grants, lifecycle leases, deletion/account guard, cleanup command; optional strict Web console and bounded mock parity | Yes, when enabled | It is user-scoped and independent of organization. Local rooted storage is development-only; production requires explicit R2. It deliberately excludes public/sharing semantics, transformations, antivirus claims, multipart upload, and usage quotas. See [`contracts/ASSETS.md`](../contracts/ASSETS.md). |
| `setting` optional starter | Finite code-owned scalar catalog, app/organization/user overrides, default resolution, monotonic CAS versions, public app ETag caching, private scope isolation, value-free audit metadata, operator CLI, and transactional account cleanup; optional strict Web preferences and bounded mock parity | Yes, when enabled with `organization` | It deliberately excludes runtime definition creation, arbitrary JSON, secrets, remote feature-flag rollout, entitlements, usage limits, and notification preferences. See [`contracts/SETTINGS.md`](../contracts/SETTINGS.md). |
| `usage` optional starter | Finite code-owned user/organization metrics, exact event idempotency, UTC counters, trusted record and atomic consume seams, versioned quota overrides, retention pruning, operator CLI, account cleanup, and private read-only Web summaries | Yes, when enabled with `organization` | Safe defaults are unlimited. It rejects arbitrary metrics and dimensions, has no public ingestion endpoint, and deliberately excludes billing, plans, entitlements, provider events, and browser quota administration. See [`contracts/USAGE.md`](../contracts/USAGE.md). |
| `webhook` optional starter | Organization-owned outbound endpoints, finite event catalog, encrypted rotating secrets, Standard Webhooks signing, SSRF-safe targets, durable outbox/delivery ledger, lease-safe retries, auto-disable, replay/prune CLI, and strict manager Web UI/mock parity | Yes, when enabled with `organization` | The browser can manage and queue only the fixed test event; trusted server modules publish product events. Delivery records exclude URLs, payloads, signatures, bodies, and free-form errors. See [`contracts/WEBHOOKS.md`](../contracts/WEBHOOKS.md). |
| Web shell | Auth route group, protected console, settings page, devtools, mock BFF guardrails, i18n, typed env | Yes | Good scaffold workspace. It is intentionally replaceable and should not become a fixed downstream workspace. |
| Contracts | Global success/error envelopes, pagination, `error_code`, `request_id`, mock BFF expectations | Yes | Cross-starter endpoint contracts still need dedicated docs as new starters are added. |
| Capabilities | Crypto, ID generation, AI, workflow, events, email, storage, queue, schedule, tracing | Partly | Email has typed all-or-none config, cancellation, a provider budget, bounded responses, and PII-safe errors; notification adds durable delivery ownership. Storage has a provider-neutral object seam, rooted private local adapter, and AWS SDK Go v2 R2 adapter; asset adds business ownership. Workflow offers local sync/memory drivers plus PostgreSQL durable tasks with fenced multi-replica claims, retries, cancellation, trace propagation, and lag metrics. Capabilities remain product-neutral and are not business starters by themselves. |

## Architecture Review Findings

| Priority | Finding | Impact | Recommended slice |
|---|---|---|---|
| P3 | The bounded AI execution capability exists, but no starter owns conversations, prompts, runs, evaluations, or cost tracking. | AI-first apps still need repeated product scaffolding. | Run semantic discovery before deciding whether an `ai-workspace` is reusable enough to enter the scaffold. |
| P3 | Usage is intentionally provider-neutral and has no pricing, invoice, tax, subscription, or payment lifecycle. | Monetized products still need product-specific commercial policy before launch. | Keep `billing` optional and provider-adapted after pricing and entitlement semantics are explicit. |

## Recommended Starter Sequence

The production auth adapter plus the organization, permission, notification, asset, setting, usage,
and webhook optional starters are complete. Keep the sequence below as an ownership map; the next
undelivered boundary is an intentionally product-sensitive AI workspace.

1. `organization` optional starter
   - Uses organization as the tenant/account boundary. Workspace is a possible future child concept, not a synonym in code or contracts.
   - The delivered starter owns organizations, membership-scoped reads, request-scoped active context, settings authorization, invitation onboarding, a privacy-minimized member directory, role/removal/leave flows, atomic ownership transfer, membership audit events, and account-deletion integrity guards.
   - Web owns directory, create, URL-scoped switching, context verification, rename, member administration, invitation management/acceptance, and ownership transfer through fixed production and mock adapters.
   - The starter remains optional; organization deletion, durable email retries, and generalized permission policy belong to later domain decisions.
   - Depends on `user`, `audit`, and email.
   - Web owns the organization switcher, member list, invitation flow, and organization settings.

2. `permission` optional starter
   - Delivered as an organization-dependent optional starter across API, migrations, contracts, Web, mock, UI, tests, audit, and governance.
   - Owns access roles, exact permission grants, policy checks, and route/service guard seams without changing organization membership roles.
   - Remains optional because simple and single-user products should not pay the role-management complexity cost.

3. `notification` optional starter
   - Delivered across API persistence/publication/worker, contracts, Web adapters/mock/UI, tests, audit, deployment guidance, and governance.
   - Owns notification records, global user preferences, read state, durable delivery attempts, and the user-facing notification center.
   - Uses events and email as capabilities; keeps SMS, Slack, or provider-specific channels as future adapters, not starter vocabulary.

4. `asset` optional starter
   - Delivered across API metadata/lifecycle/inspection/cleanup, storage capability and adapters, contracts, Web adapters/mock/UI, tests, audit, deployment guidance, and governance.
   - Owns private user metadata, idempotent intents, validation, short-lived upload/download grants, staging-to-final promotion, deletion, and account integrity.
   - Keeps storage SDKs and provider keys outside feature code; local is development-only and production requires explicit R2.

5. `setting` optional starter
   - Delivered across the finite API catalog, CAS persistence, public/private HTTP contracts,
     operator CLI, audit minimization, account cleanup, strict Web adapters/mock/UI, tests, docs,
     and governance.
   - Owns scalar typed overrides at app, organization, and user scopes; notification channel
     preferences stay with `notification`, and process configuration stays with typed startup config.
   - Keeps runtime definitions, arbitrary JSON, secrets, entitlement policy, and remote feature-flag
     rollout deliberately outside the starter.
   - Remains optional while app, user, and organization scopes share an organization-dependent
     catalog and migration. A future default-ready base should first separate app/user settings
     from the organization extension, converge its integration tests on PostgreSQL, and make the
     user locale setting authoritative for browser locale behavior.

6. `usage` optional starter before `billing`
   - Delivered across a finite API catalog, exact retained idempotency, transactional counters,
     atomic consume decisions, quota CAS/tombstones, retention, operator CLI, account cleanup,
     strict private Web adapters/mock/UI, tests, docs, and governance.
   - Owns trusted user/organization facts and hard-limit decisions. API keys, AI calls, storage, and
     workflows can emit through domain seams without importing the module.
   - Keeps arbitrary analytics, browser ingestion, entitlements, prices, plans, invoices, and
     payment-provider vocabulary outside the starter.

7. `webhook` optional starter
   - Delivered across finite event definitions, encrypted endpoint secrets, SSRF-resistant target
     resolution, transaction-aware durable publication, Standard Webhooks signatures, lease/token
     completion, bounded retry and response handling, auto-disable, replay/prune operations, strict
     manager Web adapters/mock/UI, tests, docs, and governance.
   - Keeps arbitrary browser publication, inbound callbacks, runtime schemas, response bodies,
     payload inspection, custom headers/methods, and provider-specific diagnostics outside the starter.
   - Gives integration-heavy downstream apps a production-oriented outbound path while preserving
     a small `domain.WebhookPublisher` replacement seam.

8. `ai-workspace` optional starter
   - Owns prompt templates, conversation/session records, run history, cost attribution, and evaluation hooks.
   - Uses the bounded execution contract in [`../api/docs/AI.md`](../api/docs/AI.md) plus workflow, usage, and organization starters.
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
| Mock BFF behavior | Development-only mock route handlers selected through a feature resolver or guarded by `guardMockBffRoute()` when Web needs backend-free local development |
| Audit behavior | Mutating operations emit audit metadata or explicitly document why audit does not apply |
| Downstream extraction | Keep/delete/replace guidance is clear in docs and does not leak downstream product assumptions |
| Verification | Targeted API/selected-browser tests plus `make governance`; use `make check` when multiple deployable units move |

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

The AI provider execution boundary is now bounded and documented. Run a semantic discovery slice
before building `ai-workspace`: decide whether conversation, prompt, run, evaluation, and cost
attribution are genuinely reusable starter concepts or product-specific examples. Keep provider
adapters in the AI capability and consume the delivered organization, usage, asset, webhook, and
workflow seams rather than duplicating their ownership. Billing remains separate until pricing,
entitlement, invoice, tax, and provider lifecycle semantics are explicit.
