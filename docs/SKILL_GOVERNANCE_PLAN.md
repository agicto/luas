# Luas Skill Governance Plan

This plan turns the Luas Skills System into a long-running governance layer for the scaffold.
It borrows the useful shape of small, composable engineering skills while keeping Luas-specific
vocabulary, contracts, package boundaries, and verification rules as the source of truth.

Use this plan with [`CONTEXT.md`](../CONTEXT.md), [`FRAMEWORK_QUALITY_ROADMAP.md`](FRAMEWORK_QUALITY_ROADMAP.md),
and [`../.agents/skills/luas-framework-review/SKILL.md`](../.agents/skills/luas-framework-review/SKILL.md).

## Principles

- **Small and composable**: each skill owns one repeatable discipline. Avoid one large process
  skill that tries to own every decision.
- **Router plus discipline**: user-facing router skills choose the workflow; model-invoked skills
  encode reusable engineering habits.
- **Vocabulary is active**: `CONTEXT.md` is not just read at startup. When a term changes or a new
  scaffold concept appears, the skill workflow must challenge it, resolve it, and update the
  glossary or an ADR.
- **Deep modules over busy templates**: skills should push code toward small interfaces, clear
  seams, locality, leverage, and testability.
- **Verification is part of the skill**: every implementation skill must name the command or guard
  that proves its work.
- **Scaffold-first**: skills must preserve Luas as a starter kit. They should not turn examples,
  mock BFF routes, devtools, or console pages into product behavior.

## Skill Taxonomy

### User-Invoked Router Skills

These are high-level entry points. They orchestrate, ask questions, and select lower-level skills.

| Skill | Role | Status |
|---|---|---|
| `luas-framework-review` | Global scaffold review and long-running optimization router. | Existing |
| `grill-before-build` | Clarifies wide-impact or underspecified changes before implementation. | Existing |
| `pr-description-writer` | Packages a completed diff into reviewable context. | Existing |
| `contract-evolution` | Guides HTTP contract changes across `contracts/`, `api/`, Web services, and mock BFF. | Existing |
| `downstream-app-extraction` | Guides converting Luas into a downstream app by keeping starters and deleting/replacing scaffold examples. | Existing |

### Model-Invoked Discipline Skills

These are reusable habits the agent can reach for automatically.

| Skill | Role | Status |
|---|---|---|
| `verification-before-completion` | Runs static checks, tests, builds, and relevant guard scripts. | Existing |
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
   - Pair with API/Web testing skills for style and `verification-before-completion` for the broader gate.

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
- run skill validation,
- run vocabulary and package-boundary checks,
- confirm downstream extraction docs still match the current file tree,
- write a release note that separates scaffold improvements from starter behavior changes.

## Operating Cadence

- **Every implementation turn**: run the relevant verification tier and keep changes scoped.
- **Every 3-5 framework iterations**: run `luas-framework-review` and update this plan if priorities changed.
- **Every release candidate**: run the full roadmap audit and record unresolved risks.
- **Whenever a new term appears**: decide whether it belongs in `CONTEXT.md`, an ADR, a local doc, or nowhere.

## Next Recommended Slice

Continue with the next high-leverage framework review slice: keep API package boundaries at zero
baseline exceptions, keep mock BFF success/error envelopes contract-tested, then move to security
defaults or performance evidence based on the next concrete review finding.
