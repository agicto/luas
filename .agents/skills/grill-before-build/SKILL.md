---
name: grill-before-build
description: Interview the user before implementing underspecified or wide-impact changes (features, modules, APIs, contracts) that affect persistence, permissions, deployment, or user workflows.
---

# Grill Before Build

## Purpose

Prevent premature implementation. Resolve the shape of a change before writing production code.

This skill is intentionally generic. Project-specific terms such as starter, capability, module, contract, feature, role, permission, audit, tenant, or workflow must come from the repository's own `CONTEXT.md`, ADRs, and agent instructions.

## When to Use

Use this before implementation when the user asks to add or change:

- a feature, module, page, API endpoint, integration, workflow, or job
- business behavior or user-visible behavior
- an HTTP contract or cross-service interaction
- permissions, audit, persistence, seeding, deployment, or default assembly
- architecture, folder structure, generators, or agent rails

Skip only for narrow mechanical edits, obvious typo fixes, direct command requests, or implementation of an already-approved brief.

## Source Material

Before asking the user, quickly inspect available project guidance:

1. `CONTEXT.md` or `CONTEXT-MAP.md`
2. nearby `CONTEXT.md` files in the area being changed
3. relevant ADRs in `docs/adr/` or area-specific `docs/adr/`
4. root and area-specific agent instructions
5. existing contracts, examples, and similar modules

If the answer is already clear from code or docs, state the finding and move on. Do not ask questions the repository can answer.

## Workflow

Ask one question at a time. For each question:

1. give your recommended answer
2. explain the consequence in one sentence
3. wait for the user's answer unless the answer is discoverable locally

Do not start implementation until the brief at the end is coherent.

### 1. Classify the Change

Determine the change shape using the project's own vocabulary.

Questions to resolve:

- Is this frontend-only, backend-only, cross-service, infrastructure, or documentation/process?
- Which domain concept owns it?
- Is it default behavior, optional behavior, example/prototype behavior, or app-specific behavior?
- Does it change an existing public interface?

### 2. Define the Smallest Vertical Slice

Find the narrowest demoable behavior.

Questions to resolve:

- What is the first user/caller-visible behavior?
- What is explicitly out of scope for the first slice?
- What can be mocked, faked, or deferred?
- What would prove this slice works?

### 3. Locate the Contract

If the change crosses a seam, define the interface before implementation.

Questions to resolve:

- What request, event, command, or function call crosses the seam?
- What success shape, error modes, and auth requirements exist?
- Is this a new contract or a change to an existing one?
- Which caller and which provider must change together?

### 4. Resolve Product and Domain Rules

Business rules belong here, not in scattered implementation guesses.

Questions to resolve:

- Who can perform the action?
- What states are valid before and after the action?
- What edge cases should fail, and with what stable error?
- What terminology should be added or sharpened in `CONTEXT.md`?

### 5. Resolve Operational Impact

Ask only for impacts relevant to the change.

Checklist:

- auth or permission changes
- audit or event logging
- database migration or seed data
- background jobs, queues, schedules, or retries
- external services or provider adapters
- environment variables or deployment changes
- observability, metrics, or trace requirements

### 6. Choose the Feedback Loop

Define validation before writing code.

Questions to resolve:

- What test should fail first?
- What command proves the change?
- Is browser verification needed?
- Is a throwaway prototype better than production code for this question?

## Output Brief

End the grilling session with a short implementation brief:

```md
## Implementation Brief

**Change type:** frontend-only / backend-only / cross-service / infrastructure / process
**Owning concept:** project vocabulary term
**First vertical slice:** smallest demoable behavior
**Contract:** new / changed / none
**Product rules:** resolved rules and edge cases
**Operational impact:** auth, audit, persistence, jobs, env, deploy, observability
**Out of scope:** explicit non-goals
**Feedback loop:** tests and commands to run
```

Only after this brief is coherent should implementation begin.

## Documentation Side Effects

Do not create process docs just to record the interview. Update durable project docs only when the session resolves something future agents must know:

- Add or sharpen domain terms in `CONTEXT.md`.
- Offer an ADR only for decisions that are hard to reverse, surprising without context, and based on a real tradeoff.
- Update contracts when a cross-seam interface changes.

## Red Flags

Stop and keep grilling when:

- the change has no owner concept
- "just CRUD" hides permissions, audit, state, or lifecycle rules
- frontend and backend disagree on the contract
- the first slice is too large to verify in one pass
- implementation steps are clear but success criteria are not
