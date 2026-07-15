---
name: downstream-app-extraction
description: Convert Luas into a downstream app safely. Use when deleting examples, replacing mock BFF routes, rebranding console surfaces, or checking product leakage.
---

# Downstream App Extraction

## Purpose

Turn Luas into a downstream app without confusing scaffold code with product code. Use this skill to decide what to keep, delete, replace, rename, or guard when a team starts from Luas.

## Operating Modes

Pick one mode before editing:

- **Scaffold mode**: cwd is the Luas repository or `origin` points at the Luas remote. Only make scaffold-neutral changes: docs, skills, contracts, starters, examples, guardrails, or reusable defaults. Do not add downstream product names, routes, jobs, content, credentials, or deployment behavior.
- **Downstream mode**: cwd is a downstream app repository. Product-specific behavior is allowed, but keep Luas vocabulary while classifying what was inherited from the scaffold.

If a user asks for product behavior while the worktree is still Luas scaffold mode, stop before editing product files and move to the downstream repository or ask for the correct path.

## Source Material

Read these before extraction work:

1. `CONTEXT.md` for `scaffold`, `downstream app`, `starter`, `mock BFF`, `console`, `devtools`, and `example`.
2. `docs/SCAFFOLD_SURFACES.md` for the current surface catalog, downstream actions, and verification matrix.
3. `AGENTS.md`, plus `api/AGENTS.md` or `web/AGENTS.md` when touching a half.
4. `web/docs/MOCK_BFF.md` when deleting or replacing mock route handlers.
5. `api/docs/ADDING_MODULE.md` when keeping or adding backend starter-style behavior.
6. `web/docs/ADDING_FEATURE.md` when keeping or adding Web feature behavior.
7. `contracts/README.md` when real API behavior or HTTP client behavior changes.
8. `contracts/ASSETS.md`, `api/docs/ASSETS.md`, and `web/docs/ASSETS.md` when retaining or removing
   uploaded-object behavior.

## Surface Classification

Classify each touched surface before changing it.

| Surface | Keep In Luas? | Downstream Action |
|---|---|---|
| `core` | yes | keep unless the downstream app intentionally swaps infrastructure |
| default starter | yes | keep, remove, or rename by product need |
| optional starter | yes | select it with `OPTIONAL_STARTERS` and any matching `NEXT_PUBLIC_OPTIONAL_FEATURES` only when needed; keep API replicas, database jobs, and the Web build aligned |
| capability | yes | keep if reusable; configure behind product-owned settings |
| mock BFF | yes, development-only | replace with real API, delete, or keep local-only with guards |
| console | yes, replaceable | rename or redesign for product workspace needs |
| devtools | yes, internal only | delete from production app unless explicitly needed |
| example | yes, isolated | delete or replace with product feature |
| product-specific behavior | no | belongs only in downstream mode |

## Workflow

1. **Confirm repository boundary**
   - Run `pwd`, `git remote -v`, and `git status --short --branch`.
   - In scaffold mode, reject product-specific edits and keep the change reusable.
   - In downstream mode, record the downstream app name and expected product identifiers for leakage checks.

2. **Inventory inherited surfaces**
   - List API modules, Web features, mock BFF routes, console pages, devtools, examples, env vars, background jobs, and deployment branches that will change.
   - Mark each item with one classification from the table above.

3. **Choose keep/delete/replace**
   - Keep core and capabilities unless there is a clear product reason.
   - Keep default starters when they remain useful as business-ready building blocks.
   - Select retained API optional starters through the canonical catalog; do not hand-register their routes or migrations.
   - When retaining `organization`, keep `organization` as the tenant/account term unless the product
     deliberately adds a separate child concept. Preserve member IDs as membership-resource IDs,
     keep email out of the member-directory response, and run the same `OPTIONAL_STARTERS` value in
     API replicas and migration jobs before attaching product resources or permission scopes.
     Keep active organization selection request-scoped: forward exactly one `Organization-Id` from
     the current browser tab or URL, select the retained Web feature with
     `NEXT_PUBLIC_OPTIONAL_FEATURES=organization`, apply the API's `organization_context` middleware after auth,
     and authorize product data only from the typed resolved context, never the raw header. Add
     `Organization-Id` to the production CORS allow-list when the browser calls the API cross-origin.
     When retaining the same-origin adapter, keep organization selection in the URL and extend only
     explicit Route Handlers; never replace `src/server/api-adapter/` with a browser-controlled
     catch-all proxy.
   - When retaining `permission`, retain `organization` in both optional selections, extend the
     code-owned dotted permission catalog at assembly time, and authorize only from the typed active
     organization context. Keep membership roles separate from access roles and API key scopes.
     Product modules still own resource-instance policies. Never keep permission-management UI
     without API enforcement or replace exact checks with hidden buttons.
   - When retaining `notification`, keep publication inside trusted API services, derive stable
     idempotency keys from business operations, and run `notification:work` with the same database,
     email secrets, image, and `OPTIONAL_STARTERS` selection as API replicas. Keep notification
     content plain text, action URLs local, browser state user-scoped, and provider/recipient detail
     outside contracts, audit records, and logs. Define retention before enabling high-volume use.
   - When retaining `asset`, keep user ownership and lifecycle in the starter while provider byte
     operations stay behind the storage capability. Run `asset:prune` with the same database,
     provider secrets, and `OPTIONAL_STARTERS` selection as API replicas. Preserve staging-to-final
     promotion, bounded inspection, account-deletion guard, signed-grant privacy, exact R2 CORS, and
     provider lifecycle cleanup. Keep grants out of persistent browser state and never expose object
     keys, local paths, checksums, or a generic bucket API.
   - When retaining `setting`, retain `organization` in both optional selections and keep the
     catalog finite, code-owned, scalar, and typed. Extend definitions at assembly time; do not add
     HTTP definition creation or a generic JSON editor. Preserve strong `If-Match` writes,
     monotonic reset tombstones, public app ETags, private no-store responses, value-free audit
     metadata, user cleanup, and strict Web definition validation. Keep secrets, process config,
     permissions, entitlements, usage limits, and notification preferences with their owning seams.
   - When retaining `usage`, retain `organization` in both optional selections and keep metric and
     dimension catalogs finite and code-owned. Producers call the domain record/consume seams with
     stable `source + event_id`; never add a public ingestion endpoint or browser quota writer.
     Preserve safe integers, UTC periods, atomic consume decisions, durable denials, quota CAS and
     tombstones, private summaries, the 90-day receipt horizon, pruning, and user cleanup. Keep
     telemetry, rate limits, entitlements, prices, plans, invoices, and provider events outside the
     starter. Run operator commands and prune jobs with the same database, image, clock policy, and
     `OPTIONAL_STARTERS` selection as API replicas.
   - When deleting an optional starter, remove its catalog/provider contribution and owned migration/contract surfaces, then remove its name from every environment.
   - Delete examples and devtools when they no longer teach or support the downstream app.
   - Replace mock BFF routes with production endpoints or a documented same-origin adapter.
   - For auth, read `contracts/AUTHENTICATION.md`; the Web cookie contract and Go JWT contract are
     not interchangeable through a base-URL change.
   - Preserve the provider-owned auth store. Keep `client-session` unless the downstream Next.js
     server can authoritatively resolve the real session, then replace only the bootstrap adapter.
   - Rename console surfaces only in downstream mode.

4. **Preserve contracts**
   - Update `contracts/README.md` first if request or response behavior changes.
   - Keep `error_code`, `request_id`, validation errors, and pagination stable across API, Web services, and any retained mock BFF.
   - Do not share source between `api/` and `web/`; share documented contracts.

5. **Clean product leakage**
   - Search for downstream product names, old product names, deployment names, job names, remote URLs, demo credentials, and content pipeline terms.
   - In scaffold mode, product identifiers must not appear in committed files unless they are deliberately documented as placeholders.
   - In downstream mode, Luas scaffold examples should not remain as user-visible product behavior.

6. **Verify**
   - Run the narrow commands for changed surfaces and the broader tier that proves the extraction still works.
   - For mock BFF replacement, run Web type/lint/test/build and a browser or curl check for auth, CORS, credentials, and error envelopes.
   - Before committing in Luas scaffold mode, run the product leakage script with the relevant identifiers.

## Helper Script

Use `scripts/check-downstream-contamination.sh` to scan the current repository for product-specific leakage and to confirm the expected origin remote:

```bash
bash .agents/skills/downstream-app-extraction/scripts/check-downstream-contamination.sh \
  --expected-origin git@github.com:zgiai/luas.git \
  --pattern "product-name" \
  --pattern "deployment-job-name"
```

Pass the product-specific identifiers from the current task. The script intentionally has no baked-in product names.

## Verification Matrix

| Change | Verification |
|---|---|
| Scaffold-neutral docs or skills | `validate-skill.sh --all`, vocabulary check, `git diff --check` |
| API starter kept or renamed | `cd api && go test ./...` or targeted module tests |
| Web feature kept or renamed | `cd web && pnpm type-check && pnpm lint && pnpm test -- --run` |
| Mock BFF deleted or replaced | `cd web && pnpm vitest run src/test/mock-bff-route-contract.test.ts`, then adapt/delete that guard when no mock routes remain |
| Cross-boundary extraction | `make check` plus targeted searches for product and scaffold leftovers |
| Scaffold-mode commit | contamination script with expected origin and task-specific product identifiers |

## Anti-patterns

- Editing product behavior while the worktree still points at the Luas scaffold remote.
- Treating `devtools` or examples as product features.
- Shipping mock BFF routes as a production API by accident.
- Rebranding Luas scaffold docs in the source scaffold instead of in a downstream app.
- Deleting contracts because codegen or shared source feels faster.
- Leaving demo credentials, old routes, content jobs, or product remotes in Luas.

## Pair With

- `luas-framework-review` when choosing the next scaffold governance slice.
- `domain-modeling` when a surface needs a new canonical classification.
- `contract-evolution` when replacing mock BFF behavior changes HTTP contracts.
- `luas-code-review` before merging extraction diffs.
- `verification-before-completion` before reporting extraction complete.
