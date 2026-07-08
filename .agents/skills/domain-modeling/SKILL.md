---
name: domain-modeling
description: Resolve Luas vocabulary and boundary decisions. Use when naming terms, choosing starter/feature/capability/module ownership, or updating CONTEXT.md/ADRs.
---

# Domain Modeling

## Purpose

Keep Luas vocabulary deliberate. Use this skill when a new term appears, a name feels overloaded, or a change needs a clear boundary between `core`, `starter`, `optional starter`, `capability`, `feature`, `module`, `mock BFF`, `console`, `devtools`, or `example`.

## Source Material

Read these before deciding:

1. `CONTEXT.md` for canonical global vocabulary.
2. `docs/ARCHITECTURE.md` for API/Web boundaries and vertical change flow.
3. `AGENTS.md`, plus the relevant half's `AGENTS.md`.
4. Existing ADRs under `docs/adr/`, `api/docs/adr/`, or local feature/starter docs when the term touches durable architecture.
5. `contracts/README.md` when the term appears in HTTP behavior.

## Workflow

1. **Extract the candidate term**
   - Write the exact word or phrase being introduced.
   - List where it appears or will appear: code, docs, routes, configs, contracts, tests, UI, or skills.
   - Search for existing similar terms before inventing a new one.

2. **Classify the concept**
   - Existing vocabulary: use the `CONTEXT.md` term and remove the new synonym.
   - Local implementation detail: keep it local to the owning package, feature, or doc.
   - Global scaffold vocabulary: add or refine a `CONTEXT.md` entry.
   - Contract vocabulary: update `contracts/README.md` or use `contract-evolution`.
   - Durable architecture decision: write or update an ADR only when the trade-off is hard to reverse or surprising.

3. **Choose the owning category**
   - `core`: reusable runtime or infrastructure every Luas app depends on.
   - `starter`: business-ready building block shipped with the default scaffold.
   - `optional starter`: starter-quality building block not wired into the default scaffold.
   - `capability`: reusable technical integration or helper that does not own a workflow.
   - `feature`: user-facing or developer-facing vertical slice, especially downstream or product-facing work.
   - `module`: implementation unit behind a seam, not a synonym for feature.
   - `mock BFF`: development-only Web route handlers that mimic API contracts.
   - `console`: replaceable authenticated scaffold workspace.
   - `devtools`: internal playground or demonstration routes.
   - `example`: disposable demonstration code or docs.

4. **Name the seam**
   - Prefer names that describe ownership and lifecycle, not technology alone.
   - Keep Luas as a scaffold/starter kit; do not introduce product-specific downstream app terms into the root scaffold.
   - Avoid legacy names and vague buckets such as `misc`, `common`, `utils`, or `manager` unless the local code already justifies them.
   - When a name spans API and Web, document the shared contract instead of sharing source code.

5. **Update the right artifact**
   - `CONTEXT.md`: canonical term, relationship, or flagged ambiguity.
   - ADR: durable decision with context, decision, consequences, and verification impact.
   - Local doc: implementation-specific guidance that should not become global vocabulary.
   - Skill: repeated workflow or review habit that agents should apply later.
   - Tests or guard scripts: enforce the term only when drift would be costly.

6. **Verify semantic consistency**
   - Run vocabulary checks.
   - Search for old terms, synonyms, and legacy names.
   - Confirm examples and devtools stay disposable.
   - Confirm starters and capabilities do not inherit product-specific behavior by accident.

## Verification

- `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`
- `bash .agents/skills/scripts/validate-skill.sh --all` when skills changed
- `git diff --check`
- Targeted `rg` scans for the old and new terms
- `make check` when terminology changes are coupled to code or contract behavior

## Anti-patterns

- Adding a new global term without updating `CONTEXT.md`.
- Renaming code broadly before proving the concept needs a new name.
- Using `module` when the intended meaning is user-facing `feature`.
- Turning examples, devtools, mock BFF routes, or console pages into product behavior.
- Writing an ADR for a routine local naming choice.
- Letting downstream app vocabulary leak into Luas scaffold docs or defaults.

## Pair With

- `luas-framework-review` for ranking global semantic or architecture drift.
- `contract-evolution` when vocabulary appears in HTTP behavior.
- `grill-before-build` when the term affects persistence, permissions, deployment, or user workflows.
- `verification-before-completion` before reporting the modeling decision complete.
