---
name: architecture-principles
description: Core ZGO architecture vocabulary and decision rules for seams, starter boundaries, depth, and locality
version: 1.0.0
category: development
tags: [architecture, seams, starter, locality, depth]
author: ZGO Team
updated: 2026-04-27
---

# Architecture Principles

## Purpose

This skill is the top-level architecture rulebook for ZGO. Use it before creating or refactoring modules, changing default starter assembly, or introducing new interfaces.

It defines the shared language for:

- `module`
- `interface`
- `implementation`
- `seam`
- `adapter`
- `depth`
- `leverage`
- `locality`

Other skills should inherit these rules instead of redefining them.

## When to Use

- Designing or refactoring a module
- Deciding whether something is a `starter`, `optional starter`, `capability`, or `example`
- Deciding whether to add an interface
- Reviewing a module that feels too fragmented or too coupled
- Updating scaffolding, generators, or default rules used by AI agents

## Vocabulary

- **Module**: a unit with an interface and an implementation
- **Interface**: everything a caller must know to use the module correctly
- **Implementation**: the code hidden behind the interface
- **Seam**: a place where behaviour can vary without editing callers in place
- **Adapter**: a concrete implementation that plugs into a seam
- **Depth**: how much behaviour sits behind a small interface
- **Leverage**: how much capability callers gain from that depth
- **Locality**: how concentrated change and knowledge stay when behavior evolves

## Layer Model

Use the layer names from [CONTEXT.md](../../../CONTEXT.md):

- `core`: reusable runtime and infrastructure
- `starter`: default business-ready module wired into the scaffold
- `optional starter`: starter-quality module not wired by default
- `capability`: technical helper or integration, often without HTTP routes
- `example`: teaching or reference code, not default assembly

## Decision Rules

### 1. Classify before coding

Every new module starts with one question:

`Is this a starter, optional starter, capability, or example?`

Do not choose a template until that is answered.

### 2. Prefer deep modules

Prefer a small caller-facing interface that hides meaningful behavior.

Good:

- `CreateForUser(...)`
- `Validate(...)`
- `RequestPasswordReset(...)`

Weak:

- wrappers that only forward to another object
- modules that force callers to orchestrate five internal steps manually

### 3. Concrete first

Default to concrete constructors and concrete implementations.

Add an interface only when:

- behavior truly varies across a seam
- the caller benefits from a narrower view
- the dependency is truly external or cross-process

Do not create an interface only because:

- a template expects one
- a mock generator works better with one
- “service/repository should always be interfaces”

### 4. Protect locality

When a rule changes, the preferred outcome is one focused edit.

If a new abstraction makes future changes spread across many callers, it is not helping.

### 5. Apply the deletion test

Ask:

`If I delete this module, does complexity vanish, or does it spill into many callers?`

- If complexity vanishes, the module is probably shallow.
- If complexity spills into many callers, the module is earning its keep.

## Starter Rules

- Default starters should solve common first-week project needs.
- Default starters should expose self-service surfaces by default, not broad admin/control-plane APIs.
- Authentication identifiers must be unambiguous. If a starter accepts multiple login identifiers, each identifier must have clear uniqueness semantics.
- Route-owning starters should expose stable machine-readable `error_code` values for non-2xx responses. Human-readable messages can evolve; error codes are part of the contract.
- Adding a default starter should update the starter assembly seam and its contract tests.
- Product-specific modules should not be added to default starters.

## Testing Rules

- The interface is the test surface.
- Prefer black-box tests through real seams.
- Use mocks at real seams, especially external HTTP, queues, cloud SDKs, or email providers.
- Do not widen public interfaces just to satisfy mocking tools.

## Required Follow-Through

When architecture changes:

1. Update [CONTEXT.md](../../../CONTEXT.md) if vocabulary changed.
2. Update relevant ADRs in [`docs/adr/`](../../../docs/adr/).
3. Update the specific implementation skill that operationalizes the rule:
   - [coding-standards](../coding-standards/)
   - [module-creation](../module-creation/)
   - [testing-strategy](../testing-strategy/)
   - [api-development](../api-development/)

## Red Flags

Stop and reassess when you see:

- “every module must have the same files”
- “every constructor should return an interface”
- “add a mock, therefore add an interface”
- route-owning starters exposing admin or control-plane APIs by default
- multiple assembly lists describing the same default starter set

## Validation Questions

Before finishing an architecture change, ask:

- What seam got deeper?
- What caller got simpler?
- What change is now more local?
- Which skill should encode this rule so future AI output stays aligned?
