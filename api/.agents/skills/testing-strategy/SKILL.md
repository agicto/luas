---
name: testing-strategy
description: Choose Luas API test seams, doubles, and unit/integration coverage. Use when test ownership or strategy is unclear, not merely to run an existing test.
---

# API Testing Strategy

## Purpose

Choose the smallest test seam that proves API behavior without inventing
interfaces, substituting an incompatible database, or turning every package
test into an end-to-end environment.

Skip this skill when a nearby test already demonstrates the correct pattern.
Use the focused command in `api/AGENTS.md` to run an existing test.

## Decision Order

1. Name the behavior and the public seam callers use.
2. Find the nearest test for that seam and follow its package/fixture style.
3. Decide whether SQL, HTTP assembly, concurrency, or an external process is
   part of the claim.
4. Select the lowest test level that still executes that behavior.

## Test Matrix

| Behavior | Preferred proof |
|---|---|
| Pure domain rule or mapping | Table-driven package unit test |
| Service orchestration | Public service/use-case test with an existing double |
| Handler status/envelope/validation | `httptest` through the assembled handler/router seam |
| Repository query, lock, constraint, migration | Disposable PostgreSQL with an isolated schema |
| External HTTP/provider adapter | Bounded local test server or existing adapter double |
| Queue/scheduler lifecycle or race | Focused concurrency test; race tier when relevant |
| Multi-step running API scenario | Kest flow only when process-level behavior matters |

## Rules

- PostgreSQL is the only SQL compatibility target. Never add SQLite drivers,
  DSNs, fixtures, dialect branches, or tests.
- Use repository seams and test doubles already owned by production design.
  Do not create an interface only for a mock framework.
- Prefer black-box package tests when they express caller behavior; use the
  same package when internal state is the actual contract under test.
- Keep tests deterministic: inject time/randomness, isolate mutable globals,
  and clean resources through `t.Cleanup`.
- Assert stable machine behavior separately from human messages.
- Use `require` when continued execution is invalid; use `assert` for
  independent observations.
- Coverage percentages are diagnostic. Critical behavior and failure paths
  matter more than a global percentage target.

## PostgreSQL Integration

Use the repository's `LUAS_TEST_POSTGRES_DSN` pattern and create an isolated
schema per test or suite. Fail closed when a SQL-sensitive test lacks its
required disposable database; do not silently switch dialects or point at a
shared environment.

## Doubles

Mock only an existing dependency seam that is external, slow,
non-deterministic, or cross-process. Prefer a small hand-written fake for
stateful behavior and generated/testify mocks for stable interaction
contracts. Verify expectations only when call interaction is part of the
behavior.

Examples are available under `examples/`; load one only when nearby production
tests do not answer the question.

## Verification

```bash
go test ./internal/modules/<module>/...
bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 1 ./internal/modules/<module>/...
```

Use tier 2 only for a concurrency/race claim. Use the root release gate only
for a cross-boundary change or explicit release.

## Review Checklist

- The test executes the public behavior it claims to protect.
- No production interface was added solely for testing.
- SQL-sensitive behavior uses disposable PostgreSQL.
- Success, relevant failure, and boundary inputs are covered.
- Mutable state, goroutines, and resources are isolated and cleaned.
- Secrets and shared infrastructure are absent from fixtures.

## Related Skills

- `api-development`: handler and response implementation changes.
- `database-design`: query/schema ownership is undecided.
- `kest-flow`: a running multi-step API scenario is required.
- Root `systematic-debugging`: a failure has an unknown cause.
