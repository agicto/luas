---
name: domain-modeling
description: Resolve global Luas vocabulary or ownership boundaries. Use for starter/feature/capability/module classification or CONTEXT/ADR changes, not local symbol naming.
---

# Domain Modeling

## Purpose

Give one stable name and owner to concepts that cross files, halves, or future
projects. Ordinary variable, function, component, and package naming should
follow nearby code without loading this workflow.

## Workflow

1. Read the relevant section of `CONTEXT.md`.
2. Inspect the existing code/docs that already use the concept.
3. Classify its scope:
   - `core`: required runtime/infrastructure for every Luas app.
   - `starter`: reusable business starting point shipped by default.
   - `optional starter`: production-quality business starting point activated
     explicitly.
   - `capability`: technical ability without product ownership.
   - `module`: API route-owning implementation boundary.
   - `feature`: Web user-facing implementation boundary.
   - `example`: teaching/demo material, not default behavior.
   - `contract`: behavior shared across a seam.
4. Test the candidate term:
   - one meaning
   - one owner
   - clear boundary and replacement story
   - understandable without repository history
5. Search for conflicting or legacy usage.
6. Update the smallest durable authority:
   - global reusable vocabulary -> `CONTEXT.md`
   - hard-to-reverse structural tradeoff -> ADR
   - public HTTP behavior -> contract
   - local detail -> local docs/code only
7. Run `make agent-check` for agent-facing vocabulary changes.

## Decision Rules

- API `module` and Web `feature` are not synonyms.
- A capability does not become a starter because it has many files.
- A starter does not become core because it is enabled by default.
- Mock BFF is development infrastructure, not the production API.
- Console and devtools are replaceable scaffold surfaces, not product domains.
- Prefer an existing precise term over adding a near-synonym.
- Avoid speculative vocabulary for concepts with no implementation or active
  roadmap owner.

## Output

Record:

```text
term:
definition:
owner:
included:
excluded:
replacement/removal boundary:
authority updated:
```

Use `contract-evolution` only when the decision changes public HTTP behavior.
Use `luas-framework-review` only when the user requested a broader scaffold
audit beyond this vocabulary decision.
