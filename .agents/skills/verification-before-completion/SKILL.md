---
name: verification-before-completion
description: Run the code, the tests, and the relevant tools before claiming a task is done. Use at the end of every implementation turn before reporting completion.
---

# Verification Before Completion

## Purpose

Close the loop. "I think it works" is not a deliverable. This skill turns "done" into a defensible claim by forcing concrete verification before reporting completion.

## When to Use

Run this checklist at the end of any turn where you wrote or modified code, before saying "done" / "complete" / "ready for review".

Skip for pure-documentation, pure-prose-translation, or read-only research turns.

## Verification Tiers

Pick the highest tier the change warrants. Each tier subsumes the lower tiers.

### Tier 0 — Static (always)

- Code compiles / parses without errors.
- No new lints introduced (`go vet`, `eslint`, `tsc --noEmit` as appropriate).
- Imports tidy (`goimports`, ESLint import order).
- No leftover debug prints, `fmt.Println`, `console.log`, commented-out code blocks.

### Tier 1 — Local execution (for any logic change)

- Targeted test for the change runs and passes:
  - `api/`: `go test ./<package>/...`
  - `web/`: `pnpm test -- <pattern>` or `pnpm vitest run <file>`
- If the change touches a public API, the test exercises the public API, not only internals.
- At least one error path is exercised, not only the happy path.

### Tier 2 — End-to-end (for cross-boundary changes)

- HTTP contract changes: hit the endpoint with `curl` (or REST client) and confirm shape.
- DB schema changes: run the migration up *and* down on a scratch DB.
- UI changes: load the changed page in `pnpm dev`, exercise the path, check console for errors.
- Cross-service changes: confirm both sides of the seam still build and the contract still matches.

### Tier 3 — Wide-impact (for migrations, auth, payments, permissions, defaults)

- Run the full test suite for the affected module.
- Confirm seed data still loads.
- Grep the codebase for the old symbol/path — confirm no stale references.
- Sketch the rollback plan in one paragraph.

## Reporting

State what you ran in concrete terms.

Good:

> Ran `go test ./internal/modules/user/...` (12 passing). Hit `POST /v1/users` with curl, got `201` with the expected body. Loaded `/users` in `pnpm dev`, no console errors.

Bad:

> Tests should pass. The implementation looks correct.

If you skipped a tier, say why explicitly (e.g., "Tier 2 skipped: no UI change in this turn").

## Anti-patterns

- "The build was failing for an unrelated reason so I didn't run tests." — Investigate the unrelated reason or call it out; do not silently skip.
- Claiming a test was added but not running it.
- Running the test, getting a failure, fixing the test (not the code), and reporting success.
- Verifying only the happy path.
- Saying "lint passed" without actually running the linter.
- Saying the change is "non-invasive" to skip verification. Non-invasive changes still need Tier 0.

## Helper Script

`scripts/run-tiers.sh <tier>` wraps tier 0/1/2 for the current subtree. It
auto-detects whether you're in `api/` (Go: build, golangci-lint, test, -race)
or `web/` (Node: type-check, lint, test, build). Exit code is 0 iff every
tier passes. On failure it prints the exit code, the full command log path,
and the last output lines. Set `RUN_TIERS_LOG_TAIL_LINES` to adjust the tail
length or `RUN_TIERS_LOG_DIR` to choose the log directory.

```bash
bash .agents/skills/verification-before-completion/scripts/run-tiers.sh 0
bash .agents/skills/verification-before-completion/scripts/run-tiers.sh 1
bash .agents/skills/verification-before-completion/scripts/run-tiers.sh 2
```

## Pair With

- `tdd-regression` when a bug fix, flaky behavior, or contract-sensitive change needs a red/green regression guard.
- `code-review-guide` for what *quality* looks like once verified.
- `systematic-debugging` if verification surfaces a regression.
- `grill-before-build` opens the work; this closes it.
