---
name: systematic-debugging
description: Four-phase root-cause workflow (reproduce → isolate → identify → verify) for bugs, flaky tests, and unexpected behavior. Use when something is broken and the cause is unclear.
---

# Systematic Debugging

## Purpose

Force discipline when debugging. The default failure mode is "try things until it works" — this hides the real cause and lets the bug come back. The four phases prevent that.

## When to Use

Use this skill when:

- a test fails and the cause is not obvious from the error message
- production behavior diverges from local
- a feature works in one environment but not another
- a flaky test reappears after "fixing" it
- the user says "it's broken" without a clear next step

Skip for known errors with a documented fix path.

## The Four Phases

### Phase 1: Reproduce

Get a deterministic repro before changing anything.

Questions to answer:

- What is the exact input that triggers the bug?
- What is the exact observed output vs expected output?
- Does the bug reproduce on a clean checkout / fresh DB / cold cache?
- Does it reproduce in CI as well as locally?

Output: a minimal command or test case that fails 100% of the time.

If you cannot reproduce reliably, the bug is not understood. Do not proceed to Phase 2.

### Phase 2: Isolate

Shrink the failure surface.

Strategies:

- **Binary search the diff**: `git bisect` between a known-good commit and the failing one.
- **Binary search the input**: remove half the input and see if the bug persists.
- **Binary search the code path**: comment out branches; comment out half the module imports.
- **Strip dependencies**: replace HTTP calls with mocks, DB with sqlite, OS with stub.

Output: the smallest reproducer — ideally a failing unit test or a single curl command.

### Phase 3: Identify

Find the root cause, not the proximate symptom.

For each candidate cause, write down:

- Why does this explain the observed behavior?
- What does this predict that we have not yet checked?
- What evidence would falsify this?

Use the project's existing tooling:

- `api/`: structured logs (see `logging-standards` skill), `go test -race`, pprof, `gorm.Logger` for SQL.
- `web/`: browser devtools network/console, React DevTools, Next.js build trace.

A real root cause explains *every* observed symptom. If a candidate fix doesn't explain all symptoms, it's a band-aid, not a root cause.

### Phase 4: Verify

Prove the fix works and prove it stays fixed.

Required:

1. The minimal reproducer from Phase 1 now passes.
2. A new automated test asserts the fixed behavior so regression is caught.
3. The test fails when the fix is reverted (verify by stashing the fix and re-running).

For wide-impact fixes, also verify:

- adjacent test suites still pass
- the fix does not introduce a new failure mode elsewhere (grep for similar patterns in the codebase)

## Anti-patterns

- "I'll just add a try/catch" — hides the cause, leaves bad state.
- "It works now, must have been a flake" — flakes have causes; find them.
- "I'll add a retry" — retries hide race conditions that bite in prod.
- Fixing in production code before understanding the bug. Read first, fix second.

## Closing Output

When done, summarize in this shape:

1. The minimal reproducer
2. The root cause in one sentence
3. The fix and why it addresses the cause (not just the symptom)
4. The regression test added

## Pair With

- `verification-before-completion` to confirm the fix actually holds.
- `code-review-guide` for what defensible code looks like once fixed.
