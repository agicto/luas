---
name: luas-framework-review
description: Audit Luas across security, performance, architecture, semantics, or AI workflow. Use for explicit framework-wide reviews, not routine feature work or final self-review.
---

# Luas Framework Review

## Purpose

Find and rank cross-cutting improvements to Luas as a reusable global
scaffold. This is a portfolio review workflow, not the default workflow for
implementing or reviewing one local change.

## Trigger Gate

Use this skill when the user explicitly asks for a broad framework/scaffold
audit, sustained optimization, architecture roadmap, or comparison across
multiple quality axes.

Do not use it for:

- a concrete feature, bug, endpoint, page, or migration
- ordinary naming inside one package
- automatic post-edit self-review
- selecting tests for a completed local change

Those tasks should use the owning local skill or repository guidance.

## Workflow

1. **Establish scope**
   - Inspect `git status`, the current diff, and the user's latest objective.
   - Decide which review axes are active. Do not score every axis when the
     request names one concern.

2. **Read minimum authority**
   - Vocabulary/ownership: `CONTEXT.md`.
   - Cross-cutting flow: `docs/ARCHITECTURE.md`.
   - HTTP behavior: the owning file under `contracts/`.
   - Implementation: only the affected half's `AGENTS.md` and owning docs.
   - Roadmap: read it only when ranking work across iterations.
   - ADRs: open only when the decision touches their subject.

3. **Collect evidence**
   - Prefer code, tests, command output, measurements, and current docs.
   - Distinguish measured problems from hypotheses.
   - Note unrelated dirty work and leave it untouched.

4. **Rank candidates**
   - For a broad review, produce 3-7 candidates with axis, evidence, owner,
     smallest useful slice, verification, and risk.
   - `P0`: unsafe or misleading default.
   - `P1`: global semantic, contract, or architecture drift.
   - `P2`: local design, performance, or usability friction.
   - `P3`: polish.
   - For a concrete continuation request, resume the highest-ranked unfinished
     slice instead of recreating the full inventory.

5. **Implement one slice**
   - Keep ownership at the existing seam.
   - Update vocabulary or contracts only when their public meaning changes.
   - Use `grill-before-build` only if a high-impact decision remains genuinely
     unresolved after repository discovery.

6. **Verify and report**
   - Run the narrowest owning guard and focused tests first.
   - Run `make check` once for a cross-cutting/release gate; it already includes
     governance.
   - Report evidence, changed files, checks, remaining risks, and the next
     ranked slice.

## Review Axes

| Axis | Evidence to seek |
|---|---|
| Semantics | Canonical vocabulary, one owner per concept, no overloaded contract terms |
| Architecture | Stable seams, direction of dependencies, replaceable adapters, local changes |
| Contracts | Status, envelope, `error_code`, `request_id`, pagination, mock parity |
| Security | Safe defaults, bounded inputs/resources, privacy, authorization, supply chain |
| Performance | Benchmarks, query plans, route bundles, Web Vitals, runtime measurements |
| Usability | Clear setup, predictable workflows, actionable errors, downstream extraction |
| AI workflow | Small automatic context, precise skill routing, deterministic focused checks |

## Guard Routing

The script catalog and exact trigger map live in
[`../README.md`](../README.md). Run only guards whose owned surface changed.

Common routes:

- Agent docs or skills: `make agent-check`
- HTTP contract: `check-error-contracts.py` plus the owning capability guard
- API package direction: `check-api-boundaries.sh`
- Web security/performance/UI primitives: the matching `check-web-*.py`
- CI, dependencies, or containers: the matching supply-chain guard
- Full release evidence: `make check`

Use `scripts/scaffold-architecture-report.py` only when a multi-candidate report
or diagram must persist beyond the current turn. Write temporary reports under
`$TMPDIR` unless the user requests a committed artifact.

## Boundaries

- Never share source between API and Web to avoid writing a contract.
- Never claim a performance improvement without before/after evidence.
- Never turn examples, mock BFF, console, or devtools behavior into mandatory
  product behavior.
- Never load adjacent skills solely because they appear in `Pair With`.
- Never run `make governance` immediately before `make check`.

## Related Skills

Use only when its distinct trigger is active:

- `domain-modeling`: unresolved global vocabulary or ownership.
- `contract-evolution`: a public HTTP contract changes.
- `downstream-app-extraction`: product/scaffold separation.
- `luas-code-review`: an explicit diff review.
- `verification-before-completion`: focused proof after implementation.
