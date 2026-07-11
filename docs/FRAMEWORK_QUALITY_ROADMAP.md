# Luas Framework Quality Roadmap

This roadmap tracks long-running work to keep Luas professional, semantically clear, architecture-friendly, and AI-agent friendly.

Use [`../CONTEXT.md`](../CONTEXT.md) for vocabulary. Use the `luas-framework-review` skill before adding or re-ranking items.
Use [`SKILL_GOVERNANCE_PLAN.md`](SKILL_GOVERNANCE_PLAN.md) for the 30/60/90-day plan that keeps agent skills aligned with Luas vocabulary, contracts, and architecture.

## Quality Axes

| Axis | Target |
|---|---|
| Semantic clarity | Names, docs, routes, contracts, and skills use the global vocabulary consistently. |
| Architecture depth | Important behavior sits behind small, named seams with clear ownership and test surfaces. |
| Contract integrity | API and mock BFF behavior share status codes, response envelopes, `error_code`, `request_id`, and pagination rules. |
| Security defaults | Production defaults are safe without surprising configuration gaps. |
| Performance baseline | Performance claims are backed by build output, bundle evidence, timings, or benchmarks. |
| Usability | A downstream app can delete examples, keep starters, and add features without hunting through unrelated layers. |
| AI workflows | Agents can find context, choose skills, make scoped changes, and verify outcomes without re-learning the repo each turn. |

## Current Baseline

- Global vocabulary now lives in [`../CONTEXT.md`](../CONTEXT.md).
- API and Web remain independent deployable units and share contracts, not source code.
- Error contracts have been aligned around `code`, `error_code`, `message`, optional `errors`, and optional `request_id`.
- Scaffold-level error contracts are guarded by `.agents/skills/luas-framework-review/scripts/check-error-contracts.py`, keeping `contracts/README.md`, API response constants, and Web status fallbacks aligned.
- API default HTTP guardrails now include security headers, request body limit, cooperative request timeout, production-default rate limiting, CORS, and standard `error_code` responses for body-limit, timeout, and rate-limit failures.
- Compression is intentionally not part of the default API kernel; prefer deployment/CDN compression or explicit route/starter middleware.
- API middleware ownership is now cataloged in [`../api/docs/MIDDLEWARE.md`](../api/docs/MIDDLEWARE.md).
- Web error-code vocabulary is contract-tested so `ApiErrorCode` remains server-scoped, `ClientErrorCode` remains frontend-only, and legacy underscore codes stay normalization input only.
- Web mock BFF routes are disabled in production runtime by default through `guardMockBffRoute()` and require explicit `MOCK_BFF_ENABLED=true` opt-in for demo-only deployments.
- Web mock BFF route handlers are contract-tested so every `src/app/api/**/route.ts` file calls `guardMockBffRoute()`, uses shared response helpers for success envelopes, and avoids legacy underscore-style error codes.
- Web mock BFF replacement is documented in [`../web/docs/MOCK_BFF.md`](../web/docs/MOCK_BFF.md), including production modes, deletion seams, and verification.
- Web Query/Auth providers are route-scoped: root keeps only app-wide UI context, `(auth)` owns React Query mutations, and `(protected)` owns authenticated providers.
- Web public route hydration boundaries are guarded by `src/test/public-route-boundary.test.ts`, which blocks auth, query, HTTP, mock BFF, mock session, and Zustand runtime dependencies from `(site)` routes.
- Web i18n defaults now flow through typed env config and shared locale constants instead of duplicated hardcoded values.
- Web request locale detection is isolated in `src/i18n/locale-resolution.ts` with unit tests for cookie, `Accept-Language`, and default fallback behavior.
- Web production source env access is guarded by `src/test/env-contract.test.ts`, keeping `src/config/env.ts` as the single runtime env entry point and requiring a strong `SESSION_SECRET` for production runtime mock auth.
- Root verification is split into `make governance` for scaffold guardrails and `make check` for governance plus API/Web verification tiers. CI also calls `make governance` for the root governance job. `run-tiers.sh` prints failing command exit codes, full log paths, and configurable log tails for faster repair loops.
- The root `luas-framework-review` skill now defines the long-running review loop.
- `luas-framework-review` can now generate optional HTML architecture review reports in `$TMPDIR` for multi-candidate or cross-turn recommendations.
- HTTP contract changes now have a dedicated root `contract-evolution` skill that orders changes through `contracts/`, API behavior, Web services, mock BFF behavior, and verification.
- Vocabulary and boundary decisions now have a dedicated root `domain-modeling` skill that routes new terms to `CONTEXT.md`, ADRs, local docs, skills, or nowhere.
- Luas diff review now has a dedicated root `luas-code-review` skill that separates Standards findings from Spec findings.
- Bugs and contract-sensitive regressions now have a dedicated root `tdd-regression` skill that requires a failing test before production fixes.
- Downstream extraction now has a dedicated root `downstream-app-extraction` skill with a product-leakage scan helper for keeping product behavior out of the source scaffold.
- Scaffold surfaces are cataloged in [`SCAFFOLD_SURFACES.md`](SCAFFOLD_SURFACES.md) with downstream actions and verification by surface type.
- Skill governance now has a dedicated 30/60/90-day and long-term plan in [`SKILL_GOVERNANCE_PLAN.md`](SKILL_GOVERNANCE_PLAN.md).
- High-signal docs and every non-template `SKILL.md` are guarded by `.agents/skills/luas-framework-review/scripts/check-vocabulary.sh` and CI.
- Local Markdown links across docs and agent guidance are guarded by `.agents/skills/luas-framework-review/scripts/check-doc-links.py` and CI.
- API package boundary drift is guarded by `.agents/skills/luas-framework-review/scripts/check-api-boundaries.sh`, with any current exceptions documented in [`../api/docs/PACKAGE_BOUNDARIES.md`](../api/docs/PACKAGE_BOUNDARIES.md).
- API boundary baseline exceptions are currently zero. `internal/domain` is guarded as standard-library-only, starter registry interfaces now live in `internal/starter/assembly` instead of the old top-level starter contract package, `pkg/support` no longer owns the Luas startup banner, app-specific path helpers, debug dump/timing helpers, generic manager/pipeline pattern helpers, generic control-flow/retry/Optional helpers, generic conditional wrappers, broad string/random helpers, broad collection/map helpers, or mutating dot-notation data helpers, and the remaining `pkg/support` exported surface is guarded as `Blank`, `Filled`, `DataGet`, and `DataHas`; `pkg/response` no longer imports `internal/domain`, `internal/capabilities/ai` no longer imports `internal/infra/http`, and `internal/capabilities/workflow` no longer imports `internal/infra/config`, `internal/infra/retry`, `internal/infra/schedule`, or `internal/infra/queue`.
- Branch and release governance now lives in [`BRANCHING_AND_RELEASES.md`](BRANCHING_AND_RELEASES.md): `dev` and `dev-c` are testing branches, deployment branches are CI-managed triggers, and `release/*` or accepted feature PRs are the normal path to `main`.
- Branch/release governance is guarded by `.agents/skills/luas-framework-review/scripts/check-branch-governance.sh` and CI so docs stay aligned with deployment branch mappings.
- Scaffold surface classification is guarded by `.agents/skills/luas-framework-review/scripts/check-surface-catalog.py` and CI so the catalog, glossary, and downstream extraction workflow stay aligned.
- Starter business readiness is now reviewed in [`STARTER_BUSINESS_ROADMAP.md`](STARTER_BUSINESS_ROADMAP.md), which separates the ready-to-use default starters from planned optional starters such as organization, permission, notification, file/asset, settings, usage, billing, webhook, and AI workspace.

## Candidate Queue

### P1 — Security Defaults

Problem: API security middleware now has default guardrails and ownership docs, but future changes still need to keep the catalog, kernel tests, and production knobs in sync.

Recommended slice:

1. Keep [`../api/docs/MIDDLEWARE.md`](../api/docs/MIDDLEWARE.md) as the source of truth when moving middleware between default, starter-owned, opt-in, and deployment-owned categories.
2. Add production configuration checks when new middleware knobs are introduced.
3. Keep default kernel tests in sync when guardrails change.

Verification:

- `cd api && go test ./internal/bootstrap/... ./internal/infra/middleware/... ./internal/infra/ratelimit/...`
- `cd api && golangci-lint run ./...`

### P1 — Web Hydration Boundaries

Problem: root-level providers can make public pages pay for auth/query client hydration even when they do not need authenticated app state.

Recommended slice:

1. Keep Query/Auth providers route-scoped instead of returning them to root.
2. Keep `src/test/public-route-boundary.test.ts` aligned with the public route dependency boundary.
3. Split remaining root client boundaries, especially app-wide error handling and client-only analytics, where build evidence shows value.
4. Review cookie/header-driven i18n separately because it keeps routes dynamic even after provider scoping.

Verification:

- `cd web && pnpm build`
- Bundle comparison before/after.

### P1 — i18n Runtime Boundary

Problem: locale detection currently reads cookies and request headers, which preserves language switching but keeps otherwise public routes dynamic.

Recommended slice:

1. Keep `src/i18n/locales.ts`, `src/i18n/locale-resolution.ts`, `src/config/env.ts`, and i18n docs in sync when adding locales or changing locale detection.
2. Decide whether downstream apps prefer cookie/header locale detection or locale-prefixed static routes.
3. If static public routes become the goal, introduce an explicit route strategy rather than silently disabling locale switching.

Verification:

- `cd web && pnpm type-check`
- `cd web && pnpm build`

### P1 — Demo and Mock Production Guardrails

Problem: mock BFF and demo credentials are excellent for scaffold usability but must be difficult to ship accidentally as product behavior.

Recommended slice:

1. Keep new mock route handlers behind `guardMockBffRoute()`.
2. Return mock success payloads through `apiSuccessResponse()` and errors through the shared error response helpers.
3. Keep `SESSION_SECRET` production runtime requirements covered by `src/test/env-contract.test.ts` when changing mock auth or deployment behavior.
4. Run `src/test/mock-bff-route-contract.test.ts` when adding or deleting mock route handlers.
5. Keep `web/docs/MOCK_BFF.md` current when mock route handlers, demo credentials, or auth session behavior change.
6. Add production configuration tests when adding new demo-only flows.

Verification:

- `cd web && pnpm vitest run`
- `cd web && pnpm build`

### P1 — Scaffold Error Contract Drift

Problem: scaffold-level HTTP status and `error_code` behavior spans `contracts/README.md`, API response constants, Web fallback mapping, and mock BFF behavior; changing one without the others makes downstream apps branch on stale assumptions.

Recommended slice:

1. Keep `.agents/skills/luas-framework-review/scripts/check-error-contracts.py` aligned with the scaffold-level errors documented in `contracts/README.md`.
2. Add new scaffold-level `error_code` values to the contract first, then API response constants, then Web `ApiErrorCode` and status fallback behavior.
3. Keep domain-specific codes out of the scaffold-level table until they become shared HTTP contract behavior.

Verification:

- `python3 .agents/skills/luas-framework-review/scripts/check-error-contracts.py`
- `cd web && pnpm vitest run src/test/error-code-vocabulary.test.ts src/test/api-error-contract.test.ts`
- `make check`

### P1 — Branch and Release Discipline

Problem: shared testing branches are useful for many teams, but they become unsafe when unfinished work and release-ready work are mixed and then merged wholesale into `main`.

Recommended slice:

1. Keep [`BRANCHING_AND_RELEASES.md`](BRANCHING_AND_RELEASES.md) aligned with `.github/workflows/ci.yml` and `.github/workflows/sync-deploy-branches.yml`.
2. Treat `dev` and `dev-c` as mutable testing branches, not release candidates.
3. Assemble release content from `main` using `release/*`, accepted feature PRs, or explicit cherry-picks.
4. Keep deployment trigger branches mechanical and CI-owned.

Verification:

- `make check`
- `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`
- `bash .agents/skills/luas-framework-review/scripts/check-branch-governance.sh`
- Inspect `.github/workflows/sync-deploy-branches.yml` when changing branch names or environment mappings.

### P1 — Starter Business Readiness

Problem: the current default starter set is useful for auth, API keys, and audit, but most new SaaS, internal-tool, and developer-product projects also need reusable multi-user ownership, authorization, invitations, notification preferences, files, settings, usage, and integration flows.

Recommended slice:

1. Use [`STARTER_BUSINESS_ROADMAP.md`](STARTER_BUSINESS_ROADMAP.md) as the starter readiness matrix before adding new route-owning behavior.
2. Build `organization` as an optional starter first, because ownership scope affects permission, notification, file, settings, usage, billing, webhook, and AI workspace semantics.
3. Keep `permission` documented as planned optional starter behavior until a runnable module, migrations, contracts, Web feature, and tests exist.
4. Promote a starter into the default scaffold only after its deletion path, contract, security defaults, and downstream value are proven.

Verification:

- `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`
- `PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-doc-links.py`
- `make governance`

### P2 — Architecture Vocabulary Cleanup

Problem: some docs still use framework/module/feature/console language loosely.

Recommended slice:

1. Search docs and comments for flagged ambiguities from `CONTEXT.md`.
2. Update only user-facing or agent-facing text first.
3. Avoid broad code renames unless they remove real confusion.

Verification:

- `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`
- `git diff --check`
- Targeted `rg` scans for old terminology.

### P2 — Package and Seam Deepening

Problem: packages such as `support`, `utils`, `pkg/errors`, `pkg/response`, and `internal/starter/assembly` should be reviewed for shallow-module drift. The API boundary check also records current reverse-import exceptions so they can be migrated deliberately.

Recommended slice:

1. Pick one seam from [`../api/docs/PACKAGE_BOUNDARIES.md`](../api/docs/PACKAGE_BOUNDARIES.md) or the package list above.
2. Apply the deletion test.
3. Either document why the seam is valid, deepen/rename it, or remove one baseline exception.
4. For `internal/capabilities/workflow`, guard the now-clean boundary by keeping queue, retry, and scheduler primitives workflow-owned; only `internal/infra/*` compatibility packages may wrap them.
5. Keep `pkg/support` small by requiring new exported helpers to land at the starter, capability, or runtime seam that owns the behavior.

Verification:

- `bash .agents/skills/luas-framework-review/scripts/check-api-boundaries.sh`
- Targeted tests for the selected seam.
- Full affected-half type/lint/test command.

### P2 — AI Workflow Router

Problem: root, API, and Web skills are useful but can be hard to choose during broad framework work.

Recommended slice:

1. Use `luas-framework-review` as the router for global optimization.
2. Add pairings from relevant half-specific skills back to this root skill only where useful.
3. Keep descriptions short to avoid context bloat.

Verification:

- `.agents/skills/scripts/validate-skill.sh --all`
- Skill count checks in `.agents/skills/README.md`.

### P2 — Skill Governance Plan

Problem: skill semantic drift is now guarded across every non-template `SKILL.md`, but the skills still need steady cleanup as Luas vocabulary and architecture rules evolve.

Recommended slice:

1. Follow the 30-day plan in [`SKILL_GOVERNANCE_PLAN.md`](SKILL_GOVERNANCE_PLAN.md).
2. Continue cleaning Web skill terminology around `mock BFF`, `(protected)`, `console`, and feature structure.
3. Continue cleaning API skill terminology around response/domain error mapping.
4. Keep vocabulary checks aligned with newly flagged ambiguity patterns from `CONTEXT.md`.

Verification:

- `bash .agents/skills/scripts/validate-skill.sh --all`
- `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`
- `git diff --check`

### P2 — Root Governance Entry Point

Problem: Luas now has multiple root guardrails for vocabulary, docs, contracts, surfaces, branch discipline, package direction, and skill metadata; if they stay as separate remembered commands, agents and humans will eventually run only part of the governance set.

Recommended slice:

1. Keep `make governance` as the single local entry point for root guardrails.
2. Keep CI's root governance job calling `make governance` instead of duplicating the command list.
3. Keep `make check` running `make governance` before API and Web verification tiers.
4. Add new root guard scripts to `make governance` when they become stable enough for CI/local use.
5. Keep task-specific checks, such as downstream product leakage patterns, outside the default target unless they can run without product-specific input.

Verification:

- `make governance`
- `make check`

### P2 — Documentation Link Integrity

Problem: Luas relies on AGENTS files, skills, architecture docs, and feature/starter guides as navigation rails; stale local links make both humans and agents choose the wrong seam.

Recommended slice:

1. Keep `.agents/skills/luas-framework-review/scripts/check-doc-links.py` aligned with the docs and generated/vendor exclusions.
2. Run it whenever Markdown docs, skill bodies, AGENTS files, or README navigation change.
3. Prefer fixing stale links over widening exclusions, except for intentional templates and generated/vendor trees.

Verification:

- `python3 .agents/skills/luas-framework-review/scripts/check-doc-links.py`
- `bash .agents/skills/scripts/validate-skill.sh --all`
- `git diff --check`

### P2 — Downstream Extraction Guardrails

Problem: Luas now has a downstream extraction workflow, but future scaffold changes still need to keep examples, devtools, mock BFF routes, console surfaces, and product-specific behavior clearly classified.

Recommended slice:

1. Keep [`SCAFFOLD_SURFACES.md`](SCAFFOLD_SURFACES.md), `downstream-app-extraction`, `CONTEXT.md`, `web/docs/MOCK_BFF.md`, `api/docs/ADDING_MODULE.md`, and `web/docs/ADDING_FEATURE.md` aligned.
2. Run the product-leakage helper with task-specific identifiers before committing scaffold-mode changes that touched downstream examples or docs.
3. Keep surface classification checks in CI; avoid baking product names into Luas.

Verification:

- `python3 .agents/skills/luas-framework-review/scripts/check-surface-catalog.py`
- `bash .agents/skills/scripts/validate-skill.sh --all`
- `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`
- `bash .agents/skills/downstream-app-extraction/scripts/check-downstream-contamination.sh --expected-origin git@github.com:zgiai/luas.git --pattern "<task-product-identifier>"`

### P2 — Architecture Review Reports

Problem: architecture report generation now exists, but it should stay optional and lightweight so broad reviews do not become process-heavy.

Recommended slice:

1. Use the report helper when a review compares multiple architecture candidates or needs cross-turn continuity.
2. Keep generated reports in `$TMPDIR` unless the user explicitly wants a committed artifact.
3. Iterate the report fields only when real review runs reveal missing evidence.

Verification:

- generate a sample report in `$TMPDIR`
- `bash .agents/skills/scripts/validate-skill.sh --all`
- `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`

## Iteration Rules

- Pick one candidate per turn unless the changes are purely documentary and tightly related.
- Update `CONTEXT.md` when new vocabulary is introduced.
- Update `contracts/README.md` before changing API/Web contract behavior.
- Prefer tests at public seams over implementation-coupled tests.
- Record verification commands in the final report.
- Leave deferred candidates visible so the long-running task keeps continuity.
