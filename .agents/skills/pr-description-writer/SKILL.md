---
name: pr-description-writer
description: Turn a diff into a structured PR description (summary, motivation, test plan, risk, rollback). Use when opening a PR or asked to write commit/PR copy.
---

# PR Description Writer

## Purpose

A good PR description is the reviewer's onboarding doc and the future debugger's first clue. This skill enforces a structure that survives both.

## When to Use

- Asked to "open a PR" / "write a PR description" / "draft the PR body".
- About to call `gh pr create` and the body is empty.
- Asked to "explain this change" for review or release notes.

For single-commit PRs you may use the commit message as the body, but apply the same structure.

## Required Sections

### Summary (1–3 bullets)

What changed, in plain language. Reviewers should grok the shape in 10 seconds.

### Motivation (1 paragraph)

Why this change exists. Link the issue / incident / discussion. State the problem before the solution. If the answer is "because it's correct" — say so and link the contract.

### What Changed (bullets)

Concrete list of files / modules / behaviors touched. Group by area when the diff spans multiple subtrees (`api/`, `web/`, `contracts/`).

### Test Plan (checklist)

How you verified, and how the reviewer can verify:

```markdown
- [x] Unit: `go test ./internal/modules/X/...` — 14 passing
- [x] Integration: hit `POST /v1/X` with curl, response asserted
- [ ] Manual: load /X page, confirm Y behavior   ← for reviewer
```

Use `[x]` for steps you completed, `[ ]` for steps the reviewer should run.

### Risk & Rollback

State the blast radius and the path back.

- **Blast radius**: who is affected if this is wrong? (single endpoint, single page, whole module, all users)
- **Rollback**: how to undo? (revert commit, feature flag off, DB rollback migration)

For low-risk changes write "Low — pure refactor" and stop.

## Optional Sections (use only when relevant)

- **Screenshots / Recordings**: required for any UI change.
- **Migration Notes**: required when the change includes a DB migration; reference the `sql-migration-review` skill.
- **Breaking Changes**: required when a public API / contract / config key changes.
- **Follow-ups**: known work intentionally deferred; link issues.

## Source Material

Before writing, gather:

1. `git diff main...HEAD` for the actual change.
2. `git log main..HEAD` for the commit narrative.
3. Linked issue / spec / Linear / Slack thread from the user's request.
4. Adjacent ADRs (`docs/adr/`) if the change touches architecture.

Do not invent rationale not present in code, commits, or the user's intent.

## Style

- Lead with what changed, not the implementation.
- Use the active voice: "Adds the X handler" not "The X handler has been added".
- Mention specific file paths when they aid review (`api/internal/modules/user/handler.go`).
- Keep total length under one screen unless the change is genuinely large.
- Use the project's PR template if `.github/pull_request_template.md` or `api/.agents/skills/code-review-guide/templates/pull-request-template.md` exists — match that shape.

## Anti-patterns

- "Various improvements" — be specific.
- "See diff" — the diff is what you're describing; describe it.
- Restating every code change line-by-line — that's what the diff is for.
- Marketing language ("this awesome feature unlocks…") — keep it neutral.
- Listing every commit message — synthesize, don't dump.

## Pair With

- `code-review-guide` for the reviewer's perspective on the same diff.
- `verification-before-completion` so the Test Plan section reflects what you actually ran.
