---
name: grill-before-build
description: Resolve genuinely blocking product or architecture decisions before high-impact implementation. Use only when repository discovery cannot safely answer them.
---

# Grill Before Build

## Purpose

Prevent an irreversible guess without making routine development wait for an
interview. Repository discovery and conservative defaults come first.

## Trigger Gate

Use this skill only when all are true:

1. The change affects persistence, authorization, a public contract,
   deployment, destructive lifecycle behavior, or a major user workflow.
2. A material choice is not answered by code, tests, contracts, docs, or the
   user's existing instructions.
3. Different answers would produce meaningfully incompatible implementations.

Do not trigger for:

- a clear request with established repository patterns
- a reversible local implementation slice
- typo, refactor, test, documentation, or direct command work
- a decision already made in the current thread
- details that can be discovered from the repository

User authorization to work autonomously does not remove safety boundaries, but
it does mean non-blocking choices should be made with best judgment.

## Discovery First

Inspect only the sources relevant to the unresolved choice:

- nearest implementation and tests
- owning `AGENTS.md`
- owning contract or architecture document
- relevant ADR, if one exists
- one comparable module or feature

If those sources answer the question, record the assumption briefly and
implement. Do not ask the user to repeat repository facts.

## Blocking Question

When input is still required:

- Ask one concise batch of at most three related questions.
- Give the recommended default and its consequence.
- Separate blocking decisions from preferences.
- Continue all independent work while waiting when possible.

Do not conduct a long one-question-at-a-time interview.

## Implementation Brief

Before editing, keep a compact internal brief:

```text
owner:
first observable slice:
public contract:
security/persistence impact:
chosen defaults:
out of scope:
proof:
```

Write a durable doc only when the resolved decision will guide future work:

- vocabulary or ownership -> `CONTEXT.md`
- public behavior -> owning contract
- hard-to-reverse tradeoff -> ADR
- local implementation detail -> code/tests, no process document

## Stop Conditions

Pause only when proceeding could cause data loss, privilege expansion,
incompatible public behavior, unsafe deployment, or an implementation that
must be discarded under another plausible answer. Otherwise choose the
smallest conservative default and continue.
