---
name: luas-code-review
description: Review a Luas diff or PR against behavior and repository standards. Use for explicit review requests, not automatic self-review after every edit.
---

# Luas Code Review

## Purpose

Find actionable bugs, regressions, security risks, contract drift, and missing
tests in a concrete diff. Keep the originating request separate from general
repository standards.

## Inputs

1. The user request, issue, or stated acceptance criteria.
2. The selected diff/commit/PR.
3. The nearest applicable `AGENTS.md`.
4. Only the contract, ADR, or skill that owns a changed boundary.

Do not preload both halves, all contracts, or `luas-framework-review` for a
local diff.

## Review Axes

### Spec

- Does the diff implement the requested behavior?
- Are error, empty, disabled, and rollback paths handled?
- Is anything unrelated or product-specific included?
- Are tests attached to the externally observable behavior?

### Standards

- Does dependency direction match the owning half?
- Do names and ownership match `CONTEXT.md` where global vocabulary is active?
- Do public HTTP semantics match the owning contract?
- Are secrets, authorization, logs, inputs, and resources handled safely?
- Is new complexity justified by a real seam?

## Workflow

1. Establish the diff base and list changed files.
2. Map each changed file to its owner and observable behavior.
3. Trace the main success path and at least one failure path.
4. Inspect tests for behavior coverage, not merely line coverage.
5. Run a focused command only when it can confirm or reject a suspected
   finding. A read-only review does not require `make check`.
6. Report findings first, ordered by severity.

Severity:

- `P0`: immediate security, data-loss, or unusable release risk.
- `P1`: likely bug, contract break, privilege issue, or major regression.
- `P2`: maintainability, resilience, performance, or test gap with concrete
  impact.
- `P3`: optional polish; omit unless it materially helps.

Each finding includes:

- priority and short title
- tight file/line reference
- failing scenario or violated contract
- smallest corrective direction

If there are no findings, say so and state any residual verification gap.
Avoid summaries before findings and avoid style comments already enforced by
formatters.

## Boundaries

- Review the diff, not the entire scaffold.
- Do not invent requirements absent from the request or authority docs.
- Do not treat examples/mock behavior as production requirements.
- Do not pair with `luas-framework-review` unless the user separately requested
  a framework-wide audit.
