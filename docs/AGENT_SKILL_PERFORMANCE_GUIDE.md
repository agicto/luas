# Agent Skill Performance Guide

This guide captures the reusable lessons from reducing agent overhead in Luas.
It is intended for repositories derived from Luas and for unrelated projects
that use `AGENTS.md` plus discoverable `SKILL.md` workflows.

## Executive Rule

Skills are exception workflows, not mandatory ceremony.

Routine work should be answerable from the nearest `AGENTS.md`, neighboring
code, and tests. Load at most one primary skill when its trigger clearly
matches. Loading no skill is the correct path for many local changes.

## Why Repositories Become Slow

Agent execution commonly slows down for five reasons:

1. `AGENTS.md` requires one skill for every task.
2. Broad descriptions such as "use for all development" create false-positive
   selection.
3. One skill automatically chains several related skills and documents.
4. Tutorial-sized skill bodies consume context before implementation begins.
5. Every commit or push runs the full repository release gate.

The command runtime may be small while the task still feels slow. Measure both
shell duration and the amount of context and workflow branching required before
the first edit.

## Recommended Routing Policy

Place a version of this policy in the root `AGENTS.md`:

```markdown
Start with the smallest context that can answer the task:

1. Inspect the current diff and nearest implementation and tests.
2. Read the nearest scoped `AGENTS.md` only for the unit being changed.
3. Open architecture, contract, or vocabulary documents only when that
   boundary is active.
4. Load at most one primary skill when its trigger clearly matches. Routine
   local work already covered by local guidance and code may load none.
5. Treat `Related Skills` and `Pair With` as navigation, not automatic chaining.

Do not preload document catalogs, all ADRs, or all skills. Expand context only
when current evidence leaves a real decision open.
```

## Skill Design Standard

Each skill should satisfy these constraints:

- Own one repeatable workflow with a clear boundary.
- Use a precise positive trigger and an important negative boundary.
- Keep `SKILL.md` under 200 lines.
- Move optional examples and tutorials to `examples/` or `references/`.
- Load those resources only when nearby production code is insufficient.
- Put deterministic repeated checks in `scripts/`.
- Avoid duplicating rules already enforced by the nearest `AGENTS.md`.
- Avoid duplicating a built-in or platform-provided skill.

Prefer metadata like:

```yaml
description: >-
  Review PostgreSQL migration rollout safety. Use for new or changed database
  migrations; do not use for ordinary repository or model edits.
```

Avoid metadata like:

```yaml
description: Use whenever writing, changing, testing, or reviewing backend code.
```

## Verification Budget

Define verification by changed surface in `AGENTS.md`. This lets routine tasks
choose proof without loading another workflow.

```markdown
- Guidance-only change: run the agent guidance validator.
- Backend package change: run targeted package tests.
- Frontend feature change: run the focused test, type check, or lint command.
- Contract change: run the owning contract guard and tests on changed sides.
- Explicit release or genuinely cross-cutting change: run the full gate once.
```

An ordinary commit or push is not a release boundary. Do not run a full gate
again when an unchanged invocation already passed in the same task.

## Browser And UI Work

Use an already available browser-control tool and the project's running server
before creating custom automation. Generate Playwright or Python scripts only
for repeatable headless checks, CI coverage, or a reproduction that cannot be
proved interactively.

UI skills should encode the product domain, not generic aesthetic ambition. An
operational console benefits from dense scanning, predictable navigation, and
restrained motion; forcing every UI task through broad design exploration adds
latency and creates inconsistent output.

## Migration Procedure

Apply this sequence to each downstream repository:

1. Inventory every discoverable `SKILL.md` by scope and line count.
2. Search `AGENTS.md` and skill bodies for `always`, `must`, `every`, automatic
   pairing, full-gate commands, and watch-mode test commands.
3. Remove the requirement to select a skill for every task.
4. Merge or delete overlapping skills; keep the owner with the narrowest clear
   boundary.
5. Rewrite descriptions with explicit use and do-not-use cases.
6. Move long examples and architecture explanations out of auto-loaded bodies.
7. Add a fast guidance-only validation command.
8. Separate focused iteration checks from governance and release gates.
9. Test representative prompts and record false-positive skill selections.
10. Measure again, then commit the policy and skill changes together.

Useful inventory commands:

```bash
find . -path '*/skills/*/SKILL.md' -print | sort
find . -path '*/skills/*/SKILL.md' -print0 | xargs -0 wc -l | sort -nr
rg -n '\b(always|every task|must load|pair with|make check|pnpm test)\b' \
  --glob 'AGENTS.md' --glob 'SKILL.md' .
```

Adapt paths and commands to the repository. Do not copy Luas-specific skill
names, package boundaries, or verification commands into a different stack.

## Acceptance Criteria

A migrated repository should meet all of these conditions:

- Routine local work can proceed with zero selected skills.
- No task implicitly loads a chain of related skills.
- Every skill has a distinct trigger and non-trigger boundary.
- Every `SKILL.md` is at most 200 lines.
- Focused checks are the default during implementation.
- The full repository gate runs only for cross-cutting changes or explicit
  releases.
- Skill validation is fast enough to run after guidance-only edits.
- Representative prompts select the expected skill or correctly select none.

For reference, the Luas review covered 32 skills, reduced active skill bodies
from approximately 3,943 to 3,411 lines, and kept its guidance validation at
3.76 seconds on the measured warm run. These figures are a baseline, not a
universal target; selection accuracy and time to first useful edit matter more
than minimizing line count alone.
