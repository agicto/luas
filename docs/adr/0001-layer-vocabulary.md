# ADR-0001: Layer Vocabulary

## Status

Accepted

## Context

The repository had inconsistent terms for framework code, starter modules, optional modules, and examples. That made architecture discussions drift and made code reviews re-open the same questions.

## Decision

Use the vocabulary defined in `CONTEXT.md`:

- `core`
- `starter`
- `capability`
- `optional starter`
- `example`
- `starter registry`

These terms are the canonical names for architecture discussions, docs, and future ADRs.

## Consequences

- Architecture reviews have a stable language.
- New modules can be classified without debating terminology each time.
- Docs and code comments should stop using mixed labels for the same concept.
