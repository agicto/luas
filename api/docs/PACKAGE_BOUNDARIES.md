# API Package Boundaries

This document is the operational index for the API package boundary rules in
[`adr/0005-package-boundaries.md`](adr/0005-package-boundaries.md).

## Import Direction

The intended direct import direction is:

```text
pkg/ <- internal/capabilities/ <- internal/infra/ <- internal/modules/
```

`cmd/`, `internal/bootstrap/`, and `internal/wiring/` are assembly layers and may wire the lower
layers together.

## Rules

- `pkg/` must stay reusable and must not import `internal/...`.
- `internal/capabilities/` may import `pkg/`, but must not import `internal/infra/` or
  `internal/modules/`.
- `internal/infra/` may import `pkg/` and `internal/capabilities/`, but must not import
  `internal/modules/`.
- `internal/modules/` are route-owning starter modules and may import all lower layers.

## Boundary Check

Run from the repository root:

```bash
bash .agents/skills/luas-framework-review/scripts/check-api-boundaries.sh
```

The check uses `go list` direct imports. It blocks new reverse imports while allowing the current
known baseline exceptions below.

## Known Baseline Exceptions

These are existing boundary debts, not new patterns to copy:

| Package | Imports | Why It Is Debt | Preferred Direction |
|---|---|---|---|
| `internal/capabilities/workflow` | `internal/infra/queue`, `internal/infra/retry`, `internal/infra/schedule` | The workflow capability is currently a facade over infra queue, retry, and scheduler primitives. | Either promote the primitives into the capability, or reclassify workflow as infra/runtime glue and expose a smaller capability-facing seam. |

## Boundary Debt Progress

- `internal/capabilities/workflow` no longer imports `internal/infra/config`. Infra assembly code now maps
  `config.Config` into workflow-owned runtime configuration before calling the capability.

## Review Rule

Do not add a new exception without also updating:

- this document,
- `.agents/skills/luas-framework-review/scripts/check-api-boundaries.sh`,
- and the rationale in an ADR or roadmap entry.
