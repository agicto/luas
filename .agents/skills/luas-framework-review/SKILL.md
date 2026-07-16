---
name: luas-framework-review
description: Review Luas as a global scaffold for security, performance, usability, semantic clarity, architecture, and AI workflows.
---

# Luas Framework Review

## Purpose

Run a repeatable framework-quality review for Luas. Use this skill to keep the scaffold professional, semantically clear, architecture-friendly, and safe to evolve over many small iterations.

## Source Material

Read these first:

1. `CONTEXT.md` for canonical vocabulary.
2. `AGENTS.md`, then the relevant half's `AGENTS.md`.
3. `docs/ARCHITECTURE.md`, `docs/BRANCHING_AND_RELEASES.md`, and `contracts/README.md`.
4. Relevant ADRs under `docs/adr/`, `api/docs/adr/`, or feature docs.
5. The current diff and verification status.

If a term in code or docs conflicts with `CONTEXT.md`, treat that as a semantic issue.

## Review Axes

Score each axis with `Strong`, `Adequate`, or `Needs work`.

- **Semantic clarity**: names match `CONTEXT.md`; no overloaded terms such as framework/scaffold, feature/module, API/mock BFF, code/error_code.
- **Architecture depth**: important behavior sits behind small, named seams; shallow pass-through modules are justified or removed.
- **Contract integrity**: HTTP status, response envelope, `error_code`, `request_id`, pagination, and mock BFF behavior match `contracts/README.md`.
- **Security defaults**: production defaults are safe for CORS, secrets, cookies, headers, auth, body size, timeouts, rate limits, and dependency risk.
- **Performance baseline**: changes are backed by build output, bundle evidence, Core Web Vitals, query/runtime timing, or benchmarks when performance is the claim.
- **Usability and AI workflows**: docs, skills, scripts, and examples help a human or agent find the right seam quickly without guessing.

## Workflow

1. **Inventory**
   - List the files or modules touched by the current thread.
   - Note any dirty worktree changes you did not make and avoid reverting them.
   - Record the latest verification command and outcome if available.

2. **Find candidates**
   - Produce 3-7 concrete improvement candidates.
   - For each candidate include: axis, files, problem, recommended slice, verification command, and risk.
   - Prefer candidates that improve future iterations: vocabulary, contracts, review rails, test seams, and safety defaults.

3. **Rank**
   - Mark each candidate `P0`, `P1`, `P2`, or `P3`.
   - `P0`: unsafe or misleading default.
   - `P1`: global semantic, contract, or architecture drift.
   - `P2`: local design or usability friction.
   - `P3`: polish or documentation clarity.

4. **Optional report mode**
   - Use `scripts/scaffold-architecture-report.py` when there are multiple architecture candidates,
     a deepening proposal needs diagrams, or the recommendation should survive across turns.
   - Include axis, files, problem, proposed deeper seam, before/after flow, test impact, risk,
     rollback, verification, and recommendation strength.
   - Write reports to `$TMPDIR` unless the user asks for a committed artifact.

5. **Select one slice**
   - Pick the highest-value candidate that can be completed and verified now.
   - If the slice changes persistence, permissions, public contracts, deployment, or user workflow, run `grill-before-build` before implementation.
   - Keep the slice small enough that rollback is obvious.

6. **Implement**
   - Update code, docs, skills, or contracts at the owning seam.
   - Do not create shared source between `api/` and `web/`; share contracts and vocabulary instead.
   - If adding a reusable process, prefer a root skill or root doc over duplicating instructions in both halves.

7. **Verify**
   - Run `verification-before-completion` for code changes.
   - Run `scripts/check-vocabulary.sh` when editing high-signal vocabulary or agent-facing docs.
   - Run `scripts/check-doc-links.py` when editing Markdown docs, skill bodies, AGENTS files, or README navigation.
   - Run `scripts/check-error-contracts.py` when changing scaffold-level HTTP status or `error_code` behavior.
   - Run `scripts/check-auth-contract-boundary.py` when changing auth paths, DTOs, session ownership, public failure semantics, abuse controls, proxy trust, adapter status, or starter readiness.
   - Run `scripts/check-rate-limit-boundary.py` when changing global/auth quotas, limiter algorithms, active-bucket resource policy, shared-store claims, or Redis rate-limit ownership.
   - Run `scripts/check-cache-boundary.py` when changing cache values, TTLs, memory capacity, atomic operations, loaders, Redis adapter ownership, or cache documentation.
   - Run `scripts/check-database-boundary.py` when changing database configuration, DSNs, pool lifecycle, GORM modes, repository query shape, or PostgreSQL performance evidence.
   - Run `scripts/check-web-performance-boundary.py` when changing root/shared Web client dependencies, route bundle budgets, analytics loading, public header controls, the Web production image, bundle evidence, Lighthouse evidence, or field-performance claims.
   - Run `scripts/check-api-key-boundary.py` when changing API key persistence, scope grammar or guards, management routes, one-time plaintext handling, or the Web API key feature.
   - Run `scripts/check-sensitive-telemetry.py` when changing request logging/tracing, logger context, exception diagnostics, SQL logging, tracing statements, or audit metadata.
   - Run `scripts/check-permission-boundary.py` when changing access-role persistence, permission keys, authorization guards, organization-scoped assignments, permission contracts, or the Web permission feature.
   - Run `scripts/check-notification-boundary.py` when changing notification publication, preferences, read state, delivery leases/retries, provider idempotency, notification contracts, or the Web notification feature.
   - Run `scripts/check-asset-boundary.py` when changing asset ownership/lifecycle, object-storage adapters, transfer grants, inspection, cleanup, asset contracts, or the Web asset feature.
   - Run `scripts/check-setting-boundary.py` when changing setting definitions, scopes, values, versions, cache/privacy behavior, account cleanup, setting contracts, or the Web setting feature.
   - Run `scripts/check-usage-boundary.py` when changing usage metrics, trusted events, counters, atomic quota decisions, retention, usage contracts, or the Web usage feature.
   - Run `scripts/check-webhook-boundary.py` when changing outbound event definitions, endpoint custody, signing, target policy, delivery leases/retries, replay, webhook contracts, or the Web webhook feature.
   - Run `scripts/check-ai-boundary.py` when changing AI provider config, request validation, transport, streaming, limits, errors, or the AI product/capability boundary.
   - Run `scripts/check-config-authority.py` when changing environment loading, typed configuration, logger startup, runtime reload behavior, or configuration cache semantics.
   - Run `scripts/check-email-boundary.py` when changing outbound email config, provider HTTP behavior, templates, or starter-owned mailer calls.
   - Run `scripts/check-ci-actions.py` when changing GitHub Actions, action versions, runner requirements, permissions, or CI package-manager setup.
   - Run `scripts/check-surface-catalog.py` when changing downstream extraction guidance or scaffold surface classifications.
   - Run `scripts/check-starter-catalog.py` when changing default/optional starter assembly, manifests, migrations, contracts, activation config, or starter guidance.
   - Run `scripts/check-api-boundaries.sh` when changing API package placement or imports.
   - Run `scripts/check-branch-governance.sh` when changing branch, release, deployment-branch, or CI workflow guidance.
   - For pure docs or skills, run `git diff --check` and the skill validator when relevant.
   - If verification fails, either fix it or state the exact blocker and command output.

8. **Report**
   - State the selected slice, files changed, verification run, and the next recommended slice.
   - Keep deferred candidates in the final answer so the long task has continuity.

## Anti-patterns

- Do not perform broad refactors without a selected slice and rollback path.
- Do not claim performance wins without measurements.
- Do not let mock BFF behavior drift from the shared envelope or its documented browser contract.
- Do not add new vocabulary without updating `CONTEXT.md`.
- Do not duplicate standards across root, `api/`, and `web/` when a single root rule is enough.
- Do not turn example or devtools code into required product behavior.

## Pair With

- `contract-evolution` for HTTP contract changes that span `contracts/`, API behavior, Web services, or mock BFF behavior.
- `downstream-app-extraction` when converting Luas into a downstream app or checking product leakage back into the scaffold.
- `domain-modeling` when a new term, naming choice, or starter/feature/capability/module boundary needs a canonical decision.
- `luas-code-review` when a diff needs separate Standards and Spec review axes.
- `tdd-regression` when a candidate is a bug, flaky behavior, or contract-sensitive regression.
- `grill-before-build` for wide-impact design decisions.
- `systematic-debugging` when verification fails and the cause is unclear.
- `verification-before-completion` before reporting any implementation complete.
- `pr-description-writer` when packaging a reviewable change.

## Helper Script

- `scripts/check-vocabulary.sh` scans high-signal docs and every non-template `SKILL.md` for known terminology drift from `CONTEXT.md`, including legacy mock/API naming, loose feature/module wording, starter/capability ambiguity, and console/dashboard ambiguity.
- `scripts/check-doc-links.py` scans local Markdown links in docs and agent guidance so navigation does not point at deleted files, renamed skills, or stale examples.
- `scripts/check-error-contracts.py` keeps `contracts/README.md`, `api/pkg/response/error_codes.go`, and `web/src/http/codes.ts` aligned for scaffold-level HTTP status and `error_code` behavior.
- `scripts/check-auth-contract-boundary.py` keeps browser session endpoints, Go opaque authentication sessions, revocation, public login failure semantics, auth abuse controls, proxy trust, the production adapter status, and starter readiness explicit.
- `scripts/check-rate-limit-boundary.py` keeps the built-in limiter atomic and cardinality-bounded, blocks inert Redis activation claims, and makes shared multi-replica enforcement an explicit adapter/deployment decision.
- `scripts/check-cache-boundary.py` keeps the optional cache byte-oriented, bounded, atomic where named, free of global lifecycle state, and explicit about Redis ownership and non-authority.
- `scripts/check-database-boundary.py` keeps database settings strict, DSNs encoded, pools bounded and timeout-aware, user pagination deterministic, and PostgreSQL profiling reproducible.
- `scripts/check-web-performance-boundary.py` keeps route budgets attached to official Next.js diagnostics, public theme/header controls lean and responsive, optional analytics deferred, and synthetic evidence distinct from field Web Vitals.
- `scripts/check-api-key-boundary.py` keeps hash-only API key persistence, atomic revoke/use writes, structured scopes, route guards, one-time plaintext, fixed Web adapter paths, and mock behavior aligned.
- `scripts/check-sensitive-telemetry.py` keeps route-template request logs, credential redaction, escaped exception diagnostics, parameterized SQL, and audit metadata privacy aligned.
- `scripts/check-permission-boundary.py` keeps access roles organization-scoped, permission keys exact, authorization fail-closed, delegated administration bounded, and API/Web/contracts aligned.
- `scripts/check-notification-boundary.py` keeps notification publication idempotent, user state private, email delivery lease-safe, provider retries stable, and API/Web/contracts aligned.
- `scripts/check-asset-boundary.py` keeps asset ownership private, object I/O bounded and provider-neutral, transfer grants ephemeral, cleanup safe, and API/Web/contracts aligned.
- `scripts/check-setting-boundary.py` keeps definitions finite and scalar, CAS/reset history monotonic, public/private caching explicit, audit values absent, and API/Web/contracts aligned.
- `scripts/check-usage-boundary.py` keeps metrics and dimensions finite, ingestion trusted, idempotency exact, consumption atomic, quota history monotonic, reads private, and API/Web/contracts aligned.
- `scripts/check-webhook-boundary.py` keeps events finite and trusted, endpoint secrets encrypted, targets SSRF-resistant, signatures exact, retries lease-safe, ledgers private, and API/Web/contracts aligned.
- `scripts/check-ai-boundary.py` keeps AI execution disabled by default, explicitly modeled, bounded, redirect-safe, timeout-aware, private, deterministic, and product-neutral.
- `scripts/check-config-authority.py` keeps runtime environment access behind `config.Config`, requires one startup snapshot for logging and Wire, and blocks removed dynamic reload/cache surfaces.
- `scripts/check-email-boundary.py` keeps email configuration, caller cancellation, provider timeout, bounded response reads, privacy rules, and user-starter mailer calls aligned.
- `scripts/check-ci-actions.py` keeps external actions pinned to reviewed full commit SHAs, blocks unsafe trigger drift, and verifies the Node 24 runner/tooling contract.
- `scripts/check-surface-catalog.py` keeps `docs/SCAFFOLD_SURFACES.md` aligned with `CONTEXT.md` and the `downstream-app-extraction` surface classification table.
- `scripts/check-starter-catalog.py` keeps additive optional starter config, manifests, migrations, contracts, roadmaps, and module-creation guidance aligned.
- `scripts/check-api-boundaries.sh` uses `go list` direct imports to block new reverse imports across `pkg/`, `internal/domain/`, `internal/capabilities/`, `internal/infra/`, and `internal/modules/` while reporting current baseline exceptions, if any. It also guards the tiny allowed export surface of `api/pkg/support`.
- `scripts/check-branch-governance.sh` keeps `docs/BRANCHING_AND_RELEASES.md` aligned with the CI-managed `dev` / `dev-c` to `deploy-dev` / `deploy-dev-c` deployment-branch mapping.
- `scripts/scaffold-architecture-report.py` creates an optional HTML architecture review report in `$TMPDIR`.
  Use it for multi-candidate or cross-turn recommendations where files, problem, deeper seam, before/after flow,
  test impact, risk, rollback, and recommendation strength should be compared as one artifact.
