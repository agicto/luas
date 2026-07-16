# AGENTS.md — Luas

Instructions for AI coding agents (Claude Code, Cursor, Windsurf, Copilot, etc.) working in this monorepo.

## What is Luas

Luas (Irish for _speed_) is a two-halves AI-era scaffold:

- **`api/`** — Go backend. Module: `github.com/zgiai/luas/api`. Gin + Wire DI + GORM, DDD-flavored modules, starter system (`user`, `apikey`, `audit`).
- **`web/`** — Next.js 16 / React 19 / TypeScript / Tailwind 4 / shadcn. Feature-first folders under `src/features/`.

The two halves are independent deployable units that share contracts, not code.

## Where to start

Read [CONTEXT.md](CONTEXT.md) first for the global Luas vocabulary. It is the source of truth for terms like `scaffold`, `starter`, `core`, `capability`, `feature`, `module`, `contract`, `mock BFF`, `console`, `error_code`, `request_id`, `performance budget`, and `field Web Vital`.

Each half has its own `AGENTS.md` with the detailed rules. Read those before editing:

- [api/AGENTS.md](api/AGENTS.md) — Go conventions, DDD layering, Wire DI, response patterns, testing
- [web/AGENTS.md](web/AGENTS.md) — Next.js patterns, feature folders, i18n, shadcn primitives

Workspace-level architecture docs:

- [CONTEXT.md](CONTEXT.md) — canonical global vocabulary for the whole scaffold
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — stable seams and vertical change flow
- [docs/BRANCHING_AND_RELEASES.md](docs/BRANCHING_AND_RELEASES.md) — branch roles, testing branches, release candidates, and deployment trigger rules
- [docs/CI.md](docs/CI.md) — workflow roles, runner contract, immutable action pins, permissions, and update procedure
- [docs/DEPENDENCY_SECURITY.md](docs/DEPENDENCY_SECURITY.md) — exact package tooling, dependency execution policy, OSV scanning, SBOM, and update governance
- [docs/CONTAINER_SECURITY.md](docs/CONTAINER_SECURITY.md) — immutable image inputs, BuildKit evidence, runtime SBOM/scan policy, and downstream signing boundary
- [docs/FRAMEWORK_QUALITY_ROADMAP.md](docs/FRAMEWORK_QUALITY_ROADMAP.md) — long-running quality roadmap for professional, semantic, architecture-friendly iteration
- [docs/STARTER_BUSINESS_ROADMAP.md](docs/STARTER_BUSINESS_ROADMAP.md) — starter readiness matrix and reusable business capability roadmap
- [docs/SKILL_GOVERNANCE_PLAN.md](docs/SKILL_GOVERNANCE_PLAN.md) — 30/60/90-day plan for keeping agent workflows aligned with vocabulary, contracts, and architecture
- [docs/SCAFFOLD_SURFACES.md](docs/SCAFFOLD_SURFACES.md) — surface classification, downstream actions, and verification matrix
- [contracts/README.md](contracts/README.md) — HTTP contracts shared by `api/` and `web/`
- [contracts/AUTHENTICATION.md](contracts/AUTHENTICATION.md) — browser auth, opaque API sessions, and production adapter ownership
- [contracts/API_KEYS.md](contracts/API_KEYS.md) — user-owned API key lifecycle, one-time plaintext, scope attenuation, and browser adapter
- [contracts/ORGANIZATIONS.md](contracts/ORGANIZATIONS.md) — optional organization activation, ownership scope, API, and deliberate deferrals
- [contracts/PERMISSIONS.md](contracts/PERMISSIONS.md) — optional access roles, exact permission checks, delegated-management limits, and guard seam
- [contracts/NOTIFICATIONS.md](contracts/NOTIFICATIONS.md) — optional user notifications, preferences, read state, durable email delivery, and browser adapter
- [contracts/ASSETS.md](contracts/ASSETS.md) — optional user assets, secure transfer grants, content inspection, cleanup, and browser adapter
- [contracts/SETTINGS.md](contracts/SETTINGS.md) — optional typed app/organization/user settings, ETags, versions, audit, and deletion
- [contracts/USAGE.md](contracts/USAGE.md) — optional trusted usage events, atomic quota decisions, private summaries, and retention
- [contracts/WEBHOOKS.md](contracts/WEBHOOKS.md) — optional outbound subscriptions, Standard Webhooks signing, durable retries, privacy, and browser adapter
- [api/docs/ADDING_MODULE.md](api/docs/ADDING_MODULE.md) — backend module checklist
- [api/docs/CONFIGURATION.md](api/docs/CONFIGURATION.md) — typed configuration authority, precedence, restart lifecycle, and secrets
- [api/docs/DATABASE.md](api/docs/DATABASE.md) — strict connection configuration, bounded pool lifecycle, query budgets, and PostgreSQL profiling
- [api/docs/CACHE.md](api/docs/CACHE.md) — bounded byte contract, explicit adapters, cache-aside loading, and non-authoritative ownership
- [api/docs/AUTHENTICATION.md](api/docs/AUTHENTICATION.md) — opaque session lifecycle, revocation, retention, and replacement
- [api/docs/AI.md](api/docs/AI.md) — bounded provider execution, streaming, error privacy, and AI product boundary
- [api/docs/EMAIL.md](api/docs/EMAIL.md) — context-aware provider delivery, timeout, privacy, and best-effort ownership boundary
- [api/docs/NOTIFICATIONS.md](api/docs/NOTIFICATIONS.md) — notification publication, lease worker, delivery privacy, and replacement boundary
- [api/docs/ASSETS.md](api/docs/ASSETS.md) — asset/object distinction, secure providers, inspection, cleanup, and replacement boundary
- [api/docs/SETTINGS.md](api/docs/SETTINGS.md) — typed catalog extension, CAS persistence, CLI, privacy, and account cleanup
- [api/docs/USAGE.md](api/docs/USAGE.md) — usage catalog extension, record/consume seams, quota CAS, CLI, retention, and replacement
- [api/docs/WEBHOOKS.md](api/docs/WEBHOOKS.md) — outbound catalog extension, encrypted secrets, durable worker, replay/prune CLI, and replacement
- [api/docs/OBSERVABILITY.md](api/docs/OBSERVABILITY.md) — request-log minimization, redaction, exception diagnostics, parameterized SQL, and audit privacy
- [api/docs/WORKFLOW.md](api/docs/WORKFLOW.md) — queue driver semantics, lifecycle, and production replacement boundary
- [api/docs/DEPLOYMENT.md](api/docs/DEPLOYMENT.md) — production image, local Compose, health, logs, and deployment ownership
- [api/docs/PERMISSIONS.md](api/docs/PERMISSIONS.md) — catalog extension, transactional authorizer semantics, and replacement boundary
- [web/docs/ADDING_FEATURE.md](web/docs/ADDING_FEATURE.md) — frontend feature checklist
- [web/docs/AUTHENTICATION.md](web/docs/AUTHENTICATION.md) — auth resolution modes, store isolation, and security boundaries
- [web/docs/API_KEYS.md](web/docs/API_KEYS.md) — API key browser modes, one-time secret handling, and downstream replacement
- [web/docs/ORGANIZATIONS.md](web/docs/ORGANIZATIONS.md) — optional browser activation, URL-scoped organization selection, adapter ownership, and replacement
- [web/docs/PERMISSIONS.md](web/docs/PERMISSIONS.md) — optional permission UI, fixed browser routes, mock parity, and removal
- [web/docs/NOTIFICATIONS.md](web/docs/NOTIFICATIONS.md) — optional notification center, adapter/mock ownership, strict parsing, and removal
- [web/docs/ASSETS.md](web/docs/ASSETS.md) — optional private asset workflow, ephemeral transfer grants, mock parity, and removal
- [web/docs/SETTINGS.md](web/docs/SETTINGS.md) — optional strict setting adapters, real preferences UI, mock parity, and removal
- [web/docs/USAGE.md](web/docs/USAGE.md) — optional strict read-only usage adapters, finite catalog UI, mock parity, and removal
- [web/docs/WEBHOOKS.md](web/docs/WEBHOOKS.md) — optional manager-only webhook adapters, one-time secrets, delivery UI, mock truthfulness, and removal
- [web/docs/MOCK_BFF.md](web/docs/MOCK_BFF.md) — replacing or deleting the development mock BFF in downstream apps
- [web/docs/PERFORMANCE.md](web/docs/PERFORMANCE.md) — route bundle budgets, synthetic evidence, field Web Vitals, and change procedure

## AI Agent Skills

The repo ships a Skills System at `.agents/skills/` (root) plus `api/.agents/skills/` and `web/.agents/skills/`. Codex CLI auto-discovers these based on cwd: the root + the matching half load when you cd into `api/` or `web/`. Each skill is a self-contained workflow loaded on demand when its description matches the task.

Root skills (apply everywhere):

| Skill | Use When |
|---|---|
| [`contract-evolution`](.agents/skills/contract-evolution/) | Changing HTTP contracts across `contracts/`, `api/`, Web services, or mock BFF behavior |
| [`downstream-app-extraction`](.agents/skills/downstream-app-extraction/) | Converting Luas into a downstream app without leaking product behavior back into the scaffold |
| [`domain-modeling`](.agents/skills/domain-modeling/) | Resolving vocabulary, naming, or `starter` / `feature` / `capability` / `module` boundaries |
| [`grill-before-build`](.agents/skills/grill-before-build/) | Request is underspecified or has wide impact (persistence, permissions, deployment, user workflows) |
| [`luas-code-review`](.agents/skills/luas-code-review/) | Reviewing a Luas diff against both standards and the originating request/spec |
| [`luas-framework-review`](.agents/skills/luas-framework-review/) | Reviewing Luas as a global scaffold for security, performance, semantics, architecture, or AI workflows |
| [`systematic-debugging`](.agents/skills/systematic-debugging/) | Bug or flaky test where the cause is not obvious |
| [`tdd-regression`](.agents/skills/tdd-regression/) | Fixing bugs, regressions, or contract drift with a failing test first |
| [`verification-before-completion`](.agents/skills/verification-before-completion/) | End-of-turn check that what you built actually runs / tests / lints |
| [`pr-description-writer`](.agents/skills/pr-description-writer/) | Drafting a PR body or commit summary |

Backend-specific skills are listed in [api/AGENTS.md](api/AGENTS.md#available-skills); frontend-specific in [web/AGENTS.md](web/AGENTS.md#ai-agent-skills). Full index at [.agents/skills/README.md](.agents/skills/README.md).

Helper scripts shipped with skills:

- `.agents/skills/verification-before-completion/scripts/run-tiers.sh <0|1|2> [scope...]` — auto-detects api/ vs web/ and runs the chosen tier.
- `.agents/skills/luas-framework-review/scripts/scaffold-architecture-report.py` — generate an optional HTML architecture review report in `$TMPDIR`.
- `.agents/skills/luas-framework-review/scripts/check-doc-links.py` — verify local Markdown links across docs and agent guidance.
- `.agents/skills/luas-framework-review/scripts/check-error-contracts.py` — verify scaffold-level HTTP status and `error_code` alignment across contracts, API, and Web.
- `.agents/skills/luas-framework-review/scripts/check-auth-contract-boundary.py` — keep Web/API auth ownership, public failure semantics, abuse controls, proxy trust, and adapter readiness explicit.
- `.agents/skills/luas-framework-review/scripts/check-rate-limit-boundary.py` — keep local rate-limit cardinality bounded, Redis claims honest, and multi-replica enforcement explicit.
- `.agents/skills/luas-framework-review/scripts/check-cache-boundary.py` — keep cache values driver-neutral, memory bounded, atomic operations honest, and client ownership explicit.
- `.agents/skills/luas-framework-review/scripts/check-database-boundary.py` — keep database settings strict, DSNs encoded, pools bounded, queries deterministic, and PostgreSQL performance evidence reproducible.
- `.agents/skills/luas-framework-review/scripts/check-web-performance-boundary.py` — keep Web route budgets executable, public controls lean and responsive, and synthetic/field claims distinct.
- `.agents/skills/luas-framework-review/scripts/check-web-ui-primitive-boundary.py` — keep shared composed controls attached to their semantic host with accessible disabled/loading behavior.
- `.agents/skills/luas-framework-review/scripts/check-permission-boundary.py` — keep access roles exact, organization-scoped, fail-closed, and aligned across API/Web/contracts.
- `.agents/skills/luas-framework-review/scripts/check-notification-boundary.py` — keep notification publication idempotent, delivery lease-safe, private, and aligned across API/Web/contracts.
- `.agents/skills/luas-framework-review/scripts/check-asset-boundary.py` — keep asset ownership, storage adapters, transfer grants, inspection, cleanup, and API/Web/contracts aligned.
- `.agents/skills/luas-framework-review/scripts/check-setting-boundary.py` — keep typed setting definitions finite, scalar, versioned, private where required, and aligned across API/Web/contracts.
- `.agents/skills/luas-framework-review/scripts/check-usage-boundary.py` — keep usage definitions finite, ingestion trusted, consumption atomic, quota history versioned, and API/Web/contracts aligned.
- `.agents/skills/luas-framework-review/scripts/check-webhook-boundary.py` — keep outbound catalogs finite, targets SSRF-safe, secrets encrypted, delivery leases idempotent, and API/Web/contracts aligned.
- `.agents/skills/luas-framework-review/scripts/check-ai-boundary.py` — keep AI execution disabled by default, explicitly modeled, bounded, redirect-safe, timeout-aware, private, and product-neutral.
- `.agents/skills/luas-framework-review/scripts/check-api-key-boundary.py` — keep API key hash-only persistence, atomic revoke/use writes, scope semantics, one-time plaintext, and Web adapter behavior aligned.
- `.agents/skills/luas-framework-review/scripts/check-sensitive-telemetry.py` — keep request logs/traces, exception diagnostics, SQL logs, logger context, and audit metadata behind one minimization/redaction boundary.
- `.agents/skills/luas-framework-review/scripts/check-config-authority.py` — keep API environment loading behind one typed startup snapshot and block misleading reload/cache surfaces.
- `.agents/skills/luas-framework-review/scripts/check-email-boundary.py` — keep outbound email context, timeout, response limits, privacy, config, and caller semantics aligned.
- `.agents/skills/luas-framework-review/scripts/check-ci-actions.py` — enforce reviewed full-SHA action pins, Node 24-compatible releases, explicit permissions, and safe workflow triggers.
- `.agents/skills/luas-framework-review/scripts/check-dependency-supply-chain.py` — enforce exact pnpm, safe resolution/build policy, lock integrity, pinned OSV assets, SBOM CI, Dependabot coverage, and expiring exceptions.
- `.agents/skills/luas-framework-review/scripts/check-container-supply-chain.py` — enforce digest-pinned production images, BuildKit material evidence, dual image smoke/scan CI, CycloneDX export, and expiring image exceptions.
- `.agents/skills/luas-framework-review/scripts/check-surface-catalog.py` — verify scaffold surface classifications stay aligned across context, docs, and downstream extraction guidance.
- `.agents/skills/luas-framework-review/scripts/check-starter-catalog.py` — verify optional starter selection, manifests, migrations, contracts, config, and AI guidance stay aligned.
- `.agents/skills/luas-framework-review/scripts/check-branch-governance.sh` — verify branch/release docs match CI-managed deployment branch mappings.
- `.agents/skills/pr-description-writer/scripts/scaffold-pr-body.sh [base]` — generate a PR body draft from `git log` + `git diff`.
- `api/.agents/skills/sql-migration-review/scripts/check-migration.sh <file>` — static checks for migration files.

`make governance` runs the root semantic, contract, docs, CI-action, dependency-supply-chain, surface, branch, package-boundary, and skill metadata guardrails. `make check` runs `make governance` plus the API and Web verification tiers.

CI enforces the canonical references via [.github/workflows/skill-self-test.yml](.github/workflows/skill-self-test.yml), [.github/workflows/ci.yml](.github/workflows/ci.yml), the dependency scan and SBOM via [.github/workflows/dependency-security.yml](.github/workflows/dependency-security.yml), and the API/Web image contracts via [.github/workflows/container.yml](.github/workflows/container.yml) and [.github/workflows/web-container.yml](.github/workflows/web-container.yml). The CI governance job calls `make governance` so local and CI guardrails share one entry point.

## Cross-cutting rules (apply everywhere)

1. **Don't cross the boundary.** API code never imports from `web/`. Web code only talks to API over HTTP. No shared codegen in this repo yet — keep contracts explicit.
2. **Preserve original module identity inside each half.** Inside `api/`, imports look like `github.com/zgiai/luas/api/internal/...`. Inside `web/`, imports look like `@/...`. Don't try to unify these.
3. **Brand strings.** The user-facing brand is **Luas** (capitalized). Lower-case `luas` only in identifiers (package names, binary names, env keys). Never re-introduce `LlamaFront`, `Hypership`, or `ZGO` into new code — those were the old names and have been deliberately cleaned up.
4. **History matters.** This repo was built by `git subtree add` from two upstreams; commits before the merge live under their original prefixes. Don't rewrite that history.

## Commands you might run

```bash
# api/
cd api && make wire && make run         # generate DI + start server
cd api && go vet ./...                  # quick correctness check
cd api && make test                     # run Go tests
cd api && make test-race-critical       # queue/worker lifecycle race gate
cd api && make benchmark-http           # compare core HTTP middleware with metrics off/on
cd api && make benchmark-rate-limit     # measure hot-bucket and bounded identity-churn limiter paths
cd api && make benchmark-cache          # measure bounded memory-cache reads and churn
cd api && make benchmark-database       # profile user list/write paths on disposable PostgreSQL
cd api && make container-check          # build and exercise the production image contract
cd web && bash scripts/verify-container.sh luas-web:container-check # build and exercise Web image
IMAGE=luas-api:container-check make container-scan # scan one built production image
cd api && make compose-check            # verify local DB, migration, readiness, and starter flow
cd api && make vuln                     # pinned reachable-vulnerability scan

# web/
cd web && corepack pnpm install         # exact package-manager install
cd web && corepack pnpm dev             # dev server with Turbopack
cd web && corepack pnpm type-check      # TypeScript check
cd web && corepack pnpm lint            # ESLint
cd web && corepack pnpm build           # production build plus route bundle budget gate
cd web && corepack pnpm bundle:analyze  # write official Turbopack bundle analysis output
cd web && docker build -t luas-web:local . # build the non-root standalone production image

# repo root
make governance                         # root semantic/contract/docs/CI/surface/branch/package/skill guardrails
make dependency-scan                    # live OSV scan for api/go.mod and web/pnpm-lock.yaml
make sbom                               # validated CycloneDX 1.5 inventory in $TMPDIR
make check                              # governance + API tests + web type/lint/test/build
```

## When in doubt

Defer to the per-half AGENTS.md. They are the source of truth for layering, naming, and patterns. This top-level file is just the entry point.
