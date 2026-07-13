---
name: luas-code-review
description: Review Luas diffs on separate Standards and Spec axes. Use for PR review, self-review, or checking implementation against requests, roadmap, contracts, and AGENTS.
---

# Luas Code Review

## Purpose

Review Luas changes without letting style compliance hide a wrong implementation. Always separate the **Standards axis** (does it follow Luas rules?) from the **Spec axis** (does it satisfy the actual request, roadmap slice, issue, or PRD?).

## Source Material

Read the smallest set that proves the review:

1. The user request, issue, roadmap item, PRD, or PR description that defines success.
2. The diff under review.
3. `CONTEXT.md`, `AGENTS.md`, and the affected half's `AGENTS.md`.
4. `contracts/README.md` for HTTP behavior.
5. Relevant skills when triggered by the diff: `contract-evolution`, `domain-modeling`, `tdd-regression`, `verification-before-completion`, API `code-review-guide`, Web perf/design/testing skills, or `sql-migration-review`.

## Workflow

1. **Establish the spec**
   - State the requested outcome in one sentence.
   - List explicit requirements, changed files, and any verification commands claimed.
   - If the request is unclear, review only what can be proven and list open questions.

2. **Inspect the diff**
   - Use `git diff`, `git diff --stat`, and targeted file reads.
   - Search for old names, old contract fields, old routes, or deleted symbols.
   - Do not revert user changes. Review what exists.

3. **Review the Standards axis**
   - Vocabulary: names match `CONTEXT.md`; no framework/scaffold, feature/module, mock BFF/API, or code/error_code drift.
   - Architecture: API and Web remain independent deployable units; contracts are shared as docs, not source.
   - Contracts: response envelope, `error_code`, `request_id`, pagination, validation, adapter mappings, and mock BFF guardrails match the owning docs under `contracts/`.
   - Security: production defaults are safe for secrets, CORS, cookies, headers, auth, body size, timeouts, and rate limits.
   - Testing: changes have verification at the public seam, not only implementation-coupled tests.
   - Workflow: skills, docs, scripts, and examples remain aligned with the current scaffold vocabulary.

4. **Review the Spec axis**
   - Confirm the implementation actually satisfies the originating request.
   - Check for omitted flows, wrong branch/source repository, wrong user journey, wrong release target, or incomplete cleanup.
   - Treat green tests as evidence only for the behavior they cover.
   - Mark indirect evidence as uncertain; do not infer completion from intent.

5. **Verify or challenge verification**
   - Prefer exact commands already run in the diff/PR.
   - If verification is missing or too narrow, name the command that would prove the claim.
   - For cross-boundary changes, expect `make check` plus targeted contract or mock BFF tests.
   - For pure docs or skills, expect `validate-skill.sh --all`, vocabulary checks, and `git diff --check`.

6. **Report findings first**
   - Lead with actionable findings ordered by severity.
   - Use `P0`, `P1`, `P2`, or `P3`.
   - Include file and line references when available.
   - Separate `Standards` findings from `Spec` findings when both exist.
   - After findings, list open questions, then a short summary and verification notes.

## Severity

- `P0`: unsafe, deploy-blocking, data-loss, auth bypass, wrong repository/branch, or broken production default.
- `P1`: contract drift, global semantic drift, architecture boundary violation, or implementation misses a core requirement.
- `P2`: local design, test, usability, or maintainability issue that should be fixed before broad reuse.
- `P3`: polish, wording, or low-risk documentation clarity.

## Output Shape

Use this shape for reviews:

```markdown
**Findings**
- [P1][Spec] file:line - The implementation does X, but the request requires Y. Impact...
- [P2][Standards] file:line - This introduces a new term not present in CONTEXT.md. Impact...

**Open Questions**
- ...

**Summary**
Short neutral summary of what the diff changes.

**Verification**
Commands reviewed or run, plus gaps.
```

If there are no findings, say so clearly and still mention residual risk or test gaps.

## Anti-patterns

- Merging Standards and Spec into one vague "looks good" judgment.
- Accepting green tests as proof of requirements they do not cover.
- Reviewing only changed code while ignoring contracts, docs, skills, or mock BFF behavior that should change with it.
- Treating a downstream product request as a Luas scaffold change without checking repository boundaries.
- Ending with a large summary before listing findings.

## Pair With

- `luas-framework-review` for broad scaffold quality ranking.
- `contract-evolution` for HTTP contract-sensitive diffs.
- `domain-modeling` for naming and vocabulary decisions.
- `tdd-regression` when a bug fix should include a failing test before production changes.
- `verification-before-completion` to validate fixes after review findings are addressed.
- API `code-review-guide` for backend implementation details.
