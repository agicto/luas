---
name: tdd-regression
description: Use red/green/refactor for Luas bugs, regressions, flaky behavior, contract drift, or fixes that need a failing test first.
---

# TDD Regression

## Purpose

Fix Luas bugs and contract-sensitive regressions with a reproducible failing test before changing production code. This skill turns "I fixed it" into "the old failure is now guarded at the right seam."

## When to Use

Use this skill when:

- a bug report has a concrete expected behavior,
- a regression appears after refactor, dependency upgrade, or contract change,
- API and Web disagree on an HTTP envelope, `error_code`, `request_id`, pagination, or validation behavior,
- mock BFF behavior drifts from `contracts/README.md`,
- a flaky test reveals a real race or state leak,
- a fix would otherwise be hard to prove after the turn ends.

Skip this skill for pure wording changes, mechanical renames with no behavior, or exploratory debugging where the failure is not reproducible yet. Use `systematic-debugging` first when the cause or reproduction is unclear.

## Source Material

Read the smallest set that defines the failing behavior:

1. The bug report, failing command, PR comment, issue, or user request.
2. `CONTEXT.md` for vocabulary.
3. `contracts/README.md` for HTTP contract behavior.
4. The affected half's `AGENTS.md` and testing skill:
   - API: `api/.agents/skills/testing-strategy/SKILL.md`.
   - Web: `web/.agents/skills/testing-standards/SKILL.md`.
   - Browser flow: `web/.agents/skills/webapp-testing/SKILL.md`.

## Workflow

1. **State the regression**
   - Write one sentence with input, observed behavior, expected behavior, and affected public seam.
   - Name whether the seam is API handler, domain service, Web service, route handler, component, browser flow, script, or skill workflow.

2. **Add the red test**
   - Add or update the smallest automated test that fails for the current bug.
   - Prefer the public seam callers use. Do not test a private helper only because it is easier.
   - Include at least one failing/error path when the bug is contract, auth, validation, permission, or environment related.
   - Run only the targeted test first and confirm it fails for the expected reason.

3. **Make it green**
   - Change production code only after the red test exists.
   - Keep the fix narrow and aligned with existing module, feature, capability, or contract ownership.
   - Avoid widening interfaces, adding global state, or changing mock behavior without updating contracts and docs.

4. **Refactor locally**
   - Clean up names, duplicated setup, and test fixtures introduced by the fix.
   - Keep the regression assertion intact while refactoring.
   - If the fix changed vocabulary or contracts, update `CONTEXT.md`, `contracts/README.md`, or the owning docs in the same slice.

5. **Prove the guard**
   - Re-run the targeted test and the relevant verification tier.
   - For behavior that was already covered elsewhere, explain why the new or updated test catches this exact failure.
   - When practical, prove the test would fail without the fix by reverting the production hunk locally, running the targeted test, then restoring the fix.

## Test Placement Matrix

Use the highest-level test that can reproduce the bug without making the suite brittle.

| Failure Surface | Preferred Regression Test |
|---|---|
| API response envelope, `error_code`, status, `request_id` | handler or route test through `net/http/httptest`; contract helper if present |
| Domain rule or starter behavior | service or use-case test through the module's public interface |
| Persistence query or migration behavior | repository or migration test with a real lightweight database substitute |
| Web HTTP client/service normalization | `src/test/*.test.ts` around the service or normalizer |
| Web component interaction | React Testing Library test through accessible roles and labels |
| Mock BFF route behavior | route contract test plus production and same-origin guard coverage |
| Browser-only UI regression | Playwright flow from `webapp-testing` after unit coverage is not enough |
| Skill or docs guardrail | skill validator, vocabulary check, boundary script, or a small shell test |

## Contract Regression Rules

For contract-sensitive bugs:

- Update `contracts/README.md` first if the expected contract changes.
- Keep server-scoped `ApiErrorCode` and client-only `ClientErrorCode` separate.
- Keep mock BFF routes aligned with the shared envelope and their owning browser contract; disable them in production by default.
- Assert both HTTP status and response body shape.
- Assert stable machine fields (`code`, `error_code`, `errors`, `request_id`) separately from human `message` text when possible.

## Verification

Report these in the final answer:

- red command and failure reason,
- green targeted command,
- broader command from `verification-before-completion`,
- any skipped tier and why.

Typical commands:

```bash
cd api && go test ./internal/modules/<module>/...
cd api && go test ./internal/interfaces/http/...
cd web && pnpm vitest run src/test/<file>.test.ts
cd web && pnpm type-check && pnpm lint
make check
```

## Anti-patterns

- Fixing production code first, then adding a test that merely follows the fix.
- Adding snapshots for behavior that needs explicit contract assertions.
- Mocking the broken layer so the regression can no longer fail.
- Treating `dev` or manual QA as the only regression guard.
- Updating mock BFF behavior without proving the owning browser contract and any API adapter mapping still match.
- Claiming a flaky failure is fixed without repeating the targeted test enough to build confidence.

## Pair With

- `systematic-debugging` when reproduction or root cause is unclear.
- `contract-evolution` when the expected HTTP contract changes.
- `luas-code-review` to verify the fix against both Standards and Spec axes.
- `verification-before-completion` before reporting the fix complete.
- API `testing-strategy` or Web `testing-standards` for test style in the affected half.
