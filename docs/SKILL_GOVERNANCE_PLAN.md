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

## Skill Taxonomy

### User-Invoked Router Skills

These are high-level entry points. They orchestrate, ask questions, and select lower-level skills.

| Skill | Role | Status |
|---|---|---|
| `luas-framework-review` | Explicit global scaffold review and long-running optimization router. | Existing |
| `grill-before-build` | Resolves a blocking high-impact choice after local discovery. | Existing |
| `pr-description-writer` | Packages a completed diff into reviewable context. | Existing |
| `contract-evolution` | Guides HTTP contract changes across `contracts/`, `api/`, Web services, and mock BFF. | Existing |
| `downstream-app-extraction` | Guides converting Luas into a downstream app by keeping starters and deleting/replacing scaffold examples. | Existing |

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

Forward-test representative prompts against the 32 skill descriptions and record false-positive or
false-negative selections. Keep API package boundaries at zero baseline exceptions and preserve
mock BFF contract tests while future skills are added.
