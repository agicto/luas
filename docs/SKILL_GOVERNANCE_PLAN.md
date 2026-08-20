# Luas Skill Governance Plan

This plan turns the Luas Skills System into a long-running governance layer for the scaffold.
It borrows the useful shape of small, composable engineering skills while keeping Luas-specific
vocabulary, contracts, package boundaries, and verification rules as the source of truth.

Use this plan with [`CONTEXT.md`](../CONTEXT.md), [`FRAMEWORK_QUALITY_ROADMAP.md`](FRAMEWORK_QUALITY_ROADMAP.md),
and [`../.agents/skills/luas-framework-review/SKILL.md`](../.agents/skills/luas-framework-review/SKILL.md).

## Principles

- **Small and composable**: each skill owns one repeatable discipline. Avoid one large process
  skill that tries to own every decision.
- **At most one primary skill**: routine local work may need no skill. Select a workflow only when
  its trigger clearly matches. Related-skill links are navigation, not automatic chaining.
- **Progressive disclosure**: keep `AGENTS.md` small, keep `SKILL.md` below 200 lines, and load
  contracts, ADRs, examples, and references only when their boundary is active.
- **Vocabulary is active, not automatic**: read `CONTEXT.md` when a global term or owner changes.
  Routine local edits should follow nearby code and the nearest `AGENTS.md`.
- **Deep modules over busy templates**: skills should push code toward small interfaces, clear
  seams, locality, leverage, and testability.
- **Verification is proportionate**: every implementation skill names focused proof. Run the full
  repository gate once at an explicit release boundary, not after every local edit, commit, or push.
- **One local governance entry point**: stable root guardrails should be reachable through
  `make governance`, and `make check` should include them before API/browser verification.
- **Scaffold-first**: skills must preserve Luas as a starter kit. They should not turn examples,
  mock BFF routes, devtools, or console pages into product behavior.

## 2026-07-23 Performance Reset

The official Codex model is progressive disclosure: skill metadata is present for routing, while
the full `SKILL.md` loads only after selection. The practical bottlenecks were oversized automatic
`AGENTS.md` context, broad trigger descriptions, overlapping skills, and repeated full checks.

Measured repository changes:

| Surface | Before | After | Reduction |
|---|---:|---:|---:|
| Root + API `AGENTS.md` | 37,071 bytes | 11,384 bytes | 69.3% |
| Root + Web `AGENTS.md` | 66,650 bytes | 12,901 bytes | 80.6% |
| Active repository `SKILL.md` bodies | 289,993 bytes | 164,138 bytes | 43.4% |
| Repository skills | 36 | 32 | 11.1% |
| Full governance command | `make governance`, 8.64 s | `make governance`, 4.57 s | 47.1% |
| Agent-guidance feedback command | `make governance`, 8.64 s | `make agent-check`, 0.85 s | 90.2% |

Both root-plus-half guidance paths now stay well below Codex's current default 32,768-byte project
instruction budget. The removed skills were duplicate generic standards/review workflows, the
Web-local copy of Codex's built-in `skill-creator`, and a project overview better owned by
`web/AGENTS.md`.

The fast agent-guidance loop is `make agent-check`. `make governance` remains the complete semantic
and architecture gate, and `make check` remains the single release gate.

The portable lessons and migration recipe for downstream repositories live in
[`AGENT_SKILL_PERFORMANCE_GUIDE.md`](AGENT_SKILL_PERFORMANCE_GUIDE.md).

## 2026-08-20 Progressive Disclosure Iteration

`downstream-app-extraction` now keeps only repository mode, context routing,
the extraction workflow, completion criteria, and contamination check in its
entrypoint. Optional-starter and mock BFF/auth details live in conditional
references that load only for the active downstream slice.

| Surface | Before | After | Reduction |
|---|---:|---:|---:|
| `downstream-app-extraction/SKILL.md` | 11,736 bytes | 5,917 bytes | 49.6% |
| All active repository `SKILL.md` bodies | 144,396 bytes | 138,577 bytes | 4.0% |

The next largest entrypoint is the explicit-only `accessibility-audit` skill at
7,728 bytes. Its size no longer affects routine automatic routing; revise it
only together with the Web review-boundary cleanup so WCAG behavior is not
silently weakened.

## 2026-08-20 Authority Deduplication Iteration

API persistence skills now split steady-state PostgreSQL design from migration
rollout review. MySQL rollout advice and the stale module-local migration path
were removed; the migration checker rejects MySQL syntax and reviews PostgreSQL
index/transaction behavior. Database validation no longer forces every table
into one lifecycle-column template.

Web design skills now have mutually exclusive jobs: visual direction,
design-system implementation, UI/UX review, and explicit WCAG audit. Ordinary
UI review uses checked-in Luas authority without a network fetch. Luas-specific
form and composed-control contracts moved behind an accessibility reference.

| Entrypoint | Before | After | Reduction |
|---|---:|---:|---:|
| `database-design` | 6,285 bytes | 4,302 bytes | 31.6% |
| `sql-migration-review` | 5,613 bytes | 4,609 bytes | 17.9% |
| `accessibility-audit` | 7,728 bytes | 4,982 bytes | 35.5% |
| `ui-styling-guide` | 4,147 bytes | 3,401 bytes | 18.0% |
| All active repository `SKILL.md` bodies | 138,577 bytes | 132,936 bytes | 4.1% |

`web-design-guidelines` grew from 1,562 to 2,419 bytes because it now carries a
small deterministic Luas review rubric instead of fetching an unpinned external
rulebook. This trades 857 entrypoint bytes in an explicit-only skill for lower
latency, reproducibility, and a stable authority boundary.

## 2026-08-20 Changed-Files Verification Iteration

`make agent-check-changed` now derives its scope from the feature branch's
merge-base with `main` plus staged, unstaged, and untracked files. Markdown
links and English-source policy scan only existing changed files; vocabulary
and skill metadata remain global because they are already cheap. Deletions,
renames, or checker implementation changes automatically use the complete
scan.

Measured warm runs on this workspace:

| Command/path | Time | Files scanned |
|---|---:|---:|
| `make agent-check` baseline | 2.67 s | 195 Markdown files plus all source |
| Git-detected changed-file path | 1.08-1.50 s | 1 Markdown/source file plus global cheap guards |

The observed targeted path is 43.8-59.6% faster while the pre-merge complete
gate remains unchanged.

## 2026-08-20 Forward-Routing And Budget Iteration

The checked-in routing suite records 48 realistic prompt reviews: one positive
case for every repository skill, one near-miss for each of the 11 explicit-only
workflows, and five routine no-skill tasks. The deterministic checker validates
the fixture against current invocation policy and reports false positives,
false negatives, and non-null misroutes separately.

| Evidence | Result |
|---|---:|
| Positive coverage | 32/32 skills |
| Explicit-only boundary coverage | 11/11 skills |
| Recorded false positives / false negatives / misroutes | 0 / 0 / 0 |
| Recorded routing accuracy | 48/48 (100.0%) |
| Implicit entrypoint budget | <= 6,000 bytes |
| Explicit-only entrypoint budget | <= 7,000 bytes |
| Largest implicit entrypoint | 6,494 -> 5,978 bytes |
| Active `SKILL.md` bodies | 132,936 -> 131,541 bytes |
| Standalone routing guard, five warm runs | 0.10-0.14 s |

The recorded observations are a reviewable repository routing contract, not an
independent model benchmark. A separately authorized independent run can
replace or supplement the `observed_skill` column without changing the metric
definitions. A metadata fingerprint makes description or policy changes fail
until the prompt observations are reviewed. `make agent-check` and
`make agent-check-changed` now reject stale coverage, explicit-only leakage,
routing errors, or entrypoint budget growth.

The final governance audit also removed stale exact-wording assertions from the
database boundary guard. It now accepts equivalent PostgreSQL-only,
no-SQLite, benchmark, and performance-evidence language while still rejecting
loss of those semantics. This keeps progressive-disclosure edits from failing
only because a heading was renamed.

The Web UI primitive guard now follows the accessibility entrypoint's checked
link into `references/luas-controls.md` before asserting composed-button state.
The control contract remains enforced without copying conditional detail back
into the automatically discovered entrypoint.

Downstream setting, usage, organization, and surface-catalog guards now follow
the entrypoint's conditional references and canonical catalog instead of
requiring their detailed tables or retention rules to be duplicated in
`SKILL.md`. The guards still fail when a reference link or owned invariant is
removed.

## Skill Taxonomy

Invocation policy is now enforced through every skill's `agents/openai.yaml`.
The initial policy baseline is 21 implicitly invokable skills and 11
explicit-only workflows. Use `.agents/skills/scripts/skill-metrics.sh` to
measure this split and the context-size baseline before and after each
optimization slice.

### User-Invoked Router Skills

These are high-level entry points. They orchestrate, ask questions, and select lower-level skills.

| Skill | Role | Status |
|---|---|---|
| `luas-framework-review` | Explicit global scaffold review and long-running optimization router. | Existing |
| `grill-before-build` | Resolves a blocking high-impact choice after local discovery. | Existing |
| `pr-description-writer` | Packages a completed diff into reviewable context. | Existing |
| `contract-evolution` | Guides HTTP contract changes across `contracts/`, `api/`, Web services, and mock BFF. | Existing |
| `downstream-app-extraction` | Guides converting Luas into a downstream app by keeping starters and deleting/replacing scaffold examples. | Existing |

The explicit-only set also includes high-cost local workflows for deployment,
Kest scenarios, SQL migration review, accessibility audit, design review,
browser verification, and Web performance measurement. These remain available
through explicit `$skill-name` invocation but do not compete for routine model
selection.

### Model-Invoked Discipline Skills

These are reusable habits the agent can reach for automatically.

| Skill | Role | Status |
|---|---|---|
| `verification-before-completion` | Resolves verification scope when local guidance is insufficient. | Existing |
| `systematic-debugging` | Reproduce, isolate, identify, verify. | Existing |
| `architecture-principles` | API-side seam, depth, starter, and locality rules. | Existing |
| `api-error-handling` | Web/API error response contract and code vocabulary. | Existing |
| `environment-config` | Web env source-of-truth and typed validation. | Existing |
| `domain-modeling` | Challenges vocabulary and updates `CONTEXT.md` / ADRs when terms crystallize. | Existing |
| `luas-code-review` | Reviews diffs on separate Standards and Spec axes. | Existing |
| `tdd-regression` | Runs red/green/refactor for bugs and contract-sensitive behavior. | Existing |

## 30-Day Plan

Goal: stop skills from reintroducing old Luas vocabulary or old architecture.

1. **Clean skill semantic drift**
   - Update Web strategy/styling/testing skills to use `mock BFF`, `(protected)`, `console`, `feature`, and
     `src/features/[feature]`.
   - Update API coding skills so `pkg/response` owns transport defaults while internal adapters register
     domain-specific error mappings.
   - Verification: `bash .agents/skills/scripts/validate-skill.sh --all`,
     `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`.

2. **Expand vocabulary guardrails**
   - Status: implemented. The vocabulary checker scans every non-template `SKILL.md`.
   - Keep intentional generic phrases, such as "framework" in third-party skill references, out of
     Luas-specific forbidden checks unless they describe Luas itself.
   - Verification: `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`.

3. **Improve verification failure feedback**
   - Status: implemented. `verification-before-completion/scripts/run-tiers.sh` prints the
     failing command's exit code, full log path, and configurable log tail.
   - Keep the output compact enough for CI while preserving a path to the full command log.
   - Verification: induce or simulate a failing command and confirm useful output.

## 60-Day Plan

Goal: turn skills into cross-boundary governance, not just reminders.

1. **Add `contract-evolution`**
   - Status: implemented as a root skill.
   - Required order: update `contracts/README.md`, API behavior, Web service/client behavior, mock BFF behavior,
     and contract tests.
   - Include checklists for `error_code`, `request_id`, validation errors, pagination, and production mock guardrails.

2. **Add `domain-modeling`**
   - Status: implemented as a root skill.
   - Use it when terminology changes or when a feature/starter/capability boundary is unclear.
   - Update `CONTEXT.md` inline when a canonical term is resolved.
   - Offer ADRs only for hard-to-reverse, surprising, trade-off-driven decisions.

3. **Add `luas-code-review`**
   - Status: implemented as a root skill.
   - Standards axis: compare the diff against `CONTEXT.md`, AGENTS, contracts, package boundaries, and skills.
   - Spec axis: compare the diff against the originating request, roadmap slice, issue, or PRD.
   - Keep the two reports separate so style compliance cannot hide a wrong implementation.

4. **Add `tdd-regression`**
   - Status: implemented as a root skill.
   - Require a reproducible failing test before production changes for bugs, regressions, flaky behavior, and
     contract-sensitive fixes.
   - Load API/Web testing guidance only when local test ownership is unclear. Use
     `verification-before-completion` only when the nearest verification matrix is insufficient.

## 90-Day Plan

Goal: make architecture improvement visible and repeatable.

1. **Add architecture review reports**
   - Status: implemented as an optional `luas-framework-review` helper that writes HTML to `$TMPDIR`.
   - Use it when a review has multiple architecture candidates or needs comparable evidence across turns.
   - Each candidate includes files, problem, proposed deepening, before/after flow, test impact,
     rollback notes, and recommendation strength.

2. **Deepen remaining API boundary exceptions**
   - Status: complete for the current baseline. The workflow capability no longer imports
     `internal/infra/config`, `internal/infra/retry`, `internal/infra/schedule`, or `internal/infra/queue`.
   - Keep `api/docs/PACKAGE_BOUNDARIES.md` at zero baseline exceptions unless a new exception is
     explicitly justified in an ADR or roadmap entry.
   - Treat `internal/infra/queue` and `internal/infra/schedule` as compatibility wrappers around
     workflow-owned primitives, not as the canonical implementation home for new code.

3. **Add downstream extraction workflow**
   - Status: implemented as a root skill with a product-leakage scan helper.
   - Document how a downstream app keeps default starters, replaces mock BFF, deletes examples/devtools, and
     rebrands console surfaces.
   - Pair with Web and API verification commands.

## Long-Term Plan

### Phase 1: Guardrails

Every rule that prevents architectural drift should have a script, test, or CI check:

- vocabulary drift,
- API package direction,
- mock BFF production guard,
- env source-of-truth,
- production runtime secret gate,
- public route hydration boundary,
- branch/release workflow mapping,
- docs and skill local-link integrity,
- scaffold error contract alignment,
- scaffold surface classification,
- error-code namespace split,
- mock BFF success envelope helpers,
- skill frontmatter validity.

### Phase 2: Design Feedback

Every major refactor should explain how it improves:

- seam placement,
- module depth,
- caller leverage,
- change locality,
- public test surface.

### Phase 3: Downstream Readiness

Every scaffold surface should be classified as:

- core,
- default starter,
- optional starter,
- mock BFF,
- console,
- devtools,
- example.

Status: implemented in [`SCAFFOLD_SURFACES.md`](SCAFFOLD_SURFACES.md) and guarded by
`.agents/skills/luas-framework-review/scripts/check-surface-catalog.py`.

For each category, maintain deletion/replacement instructions and verification commands.

### Phase 4: Measured Performance

Performance claims should carry evidence:

- API timing or benchmark,
- Web production build output,
- route static/dynamic classification,
- Core Web Vitals or Playwright/browser evidence.

### Phase 5: Release Discipline

Before publishing Luas as a reusable starter kit release:

- run root `make check`,
- confirm `make governance` covers skill validation, vocabulary, doc-link, error-contract,
  surface-catalog, package-boundary, and branch/release governance checks,
- confirm downstream extraction docs still match the current file tree,
- write a release note that separates scaffold improvements from starter behavior changes.

## Operating Cadence

- **Every implementation turn**: run focused proof for changed behavior and keep changes scoped.
- **Every 3-5 framework iterations**: run `luas-framework-review` and update this plan if priorities changed.
- **Every release candidate**: run `make check` once and record unresolved risks.
- **Whenever a new term appears**: decide whether it belongs in `CONTEXT.md`, an ADR, a local doc, or nowhere.

## Next Recommended Slice

Keep the routing fixture current whenever descriptions or invocation policies
change. Add an independent model run only when delegated evaluation is
explicitly authorized; preserve API package boundaries and mock BFF contract
tests while future skills evolve.
