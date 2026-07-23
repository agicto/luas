---
name: verification-before-completion
description: Choose the narrowest checks that prove changed behavior before completion. Use after implementation; reserve make check for cross-boundary or release gates.
---

# Verification Before Completion

## Purpose

Produce concrete evidence without turning every edit into a full repository
build. Verification should scale with the changed behavior and run from
focused to broad.

Skip this skill for read-only research and prose-only responses.

## Workflow

1. List changed files and the behavior they can affect.
2. Select the smallest check that executes or validates that behavior.
3. Run static checks for each changed language boundary.
4. Run focused tests for each changed behavior, including a relevant failure
   path.
5. Widen only when the change crosses a public seam or is ready to commit.
6. Report exact commands and outcomes. State any unverified risk.

## Verification Matrix

| Change | Iteration proof | Final proof |
|---|---|---|
| Agent docs or skills | `make agent-check` | `make governance` only when a governance surface changed |
| API package | focused `go test` | API tier 1; race/container checks only for their owning boundary |
| Web feature | focused Vitest plus type/lint as relevant | Web tier 2 when production output can change |
| Shared HTTP contract | owning guard plus focused API/Web tests | `make check` |
| Cross-cutting/release | focused checks first | `make check` once |

`make check` already runs `make governance`, API checks, and Web checks. Never
run `make governance` immediately before it. Do not rerun an unchanged
successful command in the same turn.

## Tier Helper

`scripts/run-tiers.sh <tier> [scope...]` auto-detects the current half:

- API tier 0: build and lint.
- API tier 1: tier 0 plus tests for the supplied Go scopes.
- API tier 2: tier 1 plus the race detector.
- Web tier 0: type check and lint.
- Web tier 1: tier 0 plus Vitest.
- Web tier 2: tier 1 plus production build and route budgets.

Examples:

```bash
cd api
bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 1 ./internal/modules/user/...

cd web
corepack pnpm vitest run src/features/auth/auth.test.ts
bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 0
```

The helper logs full output to a temporary directory and prints a short failure
tail. Set `RUN_TIERS_LOG_TAIL_LINES` or `RUN_TIERS_LOG_DIR` only when needed.
Inside a Codex sandbox it also uses a writable temporary Go build cache, so
normal verification does not require an approval merely to update `$GOCACHE`.

## Evidence Rules

- A compile, lint, or type check does not replace a behavior test.
- A unit test does not prove a changed browser workflow or migration.
- A build does not prove runtime headers, container health, or deployment.
- A performance claim needs a comparable measurement.
- A skipped required check must include the exact blocker.
- Do not claim success from stale command output after code changed.

Use `systematic-debugging` only when a verification failure has an unclear
cause. Use `tdd-regression` when the task is itself a bug or contract
regression.
