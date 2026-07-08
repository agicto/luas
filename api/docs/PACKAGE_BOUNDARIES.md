# API Package Boundaries

This document is the operational index for the API package boundary rules in
[`adr/0005-package-boundaries.md`](adr/0005-package-boundaries.md).

## Import Direction

The intended direct import direction is:

```text
internal/domain <- internal/modules/

pkg/ <- internal/capabilities/ <- internal/infra/ <- internal/modules/
```

`cmd/`, `internal/bootstrap/`, and `internal/wiring/` are assembly layers and may wire the lower
layers together. `internal/domain/` is the framework-free starter vocabulary layer; it is consumed
by starter modules and selected assembly code, but it does not depend on Luas runtime packages.

## Rules

- `pkg/` must stay reusable and must not import `internal/...`.
- `internal/domain/` must stay framework-free and must not import `pkg/` or `internal/...`.
- `internal/capabilities/` may import `pkg/`, but must not import `internal/domain/`,
  `internal/infra/`, or `internal/modules/`.
- `internal/infra/` may import `pkg/` and `internal/capabilities/`, but must not import
  `internal/domain/` or `internal/modules/`.
- `internal/modules/` are route-owning starter modules and may import `internal/domain/` plus
  lower runtime layers when a real seam needs them.

## Boundary Check

Run from the repository root:

```bash
bash .agents/skills/luas-framework-review/scripts/check-api-boundaries.sh
```

The check uses `go list` direct imports. It blocks new reverse imports and reports the current
known baseline exception count.

## Known Baseline Exceptions

None.

## Boundary Debt Progress

- `internal/domain/` is now guarded as a standard-library-only package so domain entities,
  domain errors, and repository seams do not pick up HTTP, database, runtime, or response helpers.
- The Luas startup banner now lives in `internal/bootstrap/` instead of `pkg/support`, keeping
  application-branded console output out of reusable `pkg/` helpers.
- Starter registry interfaces now live in `internal/starter/assembly` instead of the old
  top-level starter contract package, keeping HTTP contract vocabulary reserved for `contracts/`.
- App-specific path helpers were removed from `pkg/support`; runtime-owned packages should compute
  their own paths from configuration or explicit inputs instead of using global scaffold path state.
- Debug dump, timing, stack, and memory-print helpers were removed from `pkg/support`; local
  diagnostics should live in devtools or internal runtime packages when a real caller needs them.
- Generic driver manager and pipeline pattern helpers were removed from `pkg/support`; driver
  registries, middleware chains, and workflow pipelines should live at the owning capability or
  runtime seam with domain-specific names.
- Generic control-flow, retry, panic, and Optional helpers were removed from `pkg/support`; retry
  behavior should stay at workflow or integration seams, and nil/error handling should be explicit
  at the caller or owning package.
- Generic conditional wrapper helpers were removed from `pkg/support`; conditional response,
  resource, query, or workflow behavior should live at the semantic seam that owns the decision.
- Broad string formatting, slug, random string, and UUID helpers were removed from `pkg/support`;
  string normalization, slugging, and ID generation should live at the caller or the owning
  capability, such as `internal/capabilities/idgen`.
- `internal/capabilities/workflow` no longer imports `internal/infra/config`. Infra assembly code now maps
  `config.Config` into workflow-owned runtime configuration before calling the capability.
- `internal/capabilities/workflow` no longer imports `internal/infra/retry`. Synchronous retry behavior now
  lives inside the workflow capability and is guarded by success, stop-retry, and exhaustion tests.
- `internal/capabilities/workflow` no longer imports `internal/infra/schedule`. Scheduler behavior now lives
  inside the workflow capability; `internal/infra/schedule` remains as a compatibility wrapper.
- `internal/capabilities/workflow` no longer imports `internal/infra/queue`. Background job queue behavior now
  lives inside the workflow capability; `internal/infra/queue` remains as a compatibility wrapper.

## Review Rule

Do not add a new exception without also updating:

- this document,
- `.agents/skills/luas-framework-review/scripts/check-api-boundaries.sh`,
- and the rationale in an ADR or roadmap entry.
