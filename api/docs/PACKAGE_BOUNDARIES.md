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
- Generic grab-bag packages are gone. After their helpers moved to owning seams, `pkg/support` and
  `pkg/utils` had no production callers and were removed (2026-09); the boundary check now rejects
  `support`, `utils`, `common`, and `helpers` packages under `pkg/` and `internal/infra/`.
- Credential-shaped telemetry sanitization lives in the focused, standard-library-only
  `pkg/redact` package rather than a broad support helper or duplicated logger/exception blacklists.
- Unadopted Laravel-style ports without production callers were removed (2026-09): `pkg/request`
  (form requests), `pkg/resource` (API resources), `pkg/validation`, `pkg/hash`, the deprecated
  `pkg/encryption` (superseded by `internal/capabilities/crypto`), `internal/infra/http`,
  `internal/infra/lang`, and `internal/infra/types`.
- `internal/capabilities/workflow` no longer imports `internal/infra/config`. Infra assembly code now maps
  `config.Config` into workflow-owned runtime configuration before calling the capability.
- `internal/capabilities/workflow` no longer imports `internal/infra/retry`. Synchronous retry behavior now
  lives inside the workflow capability and is guarded by success, stop-retry, and exhaustion tests.
- Queue, scheduler, and retry behavior live only in `internal/capabilities/workflow`. The former
  `internal/infra/queue`, `internal/infra/schedule`, and `internal/infra/retry` compatibility wrappers
  were removed (2026-09) and their tests now exercise the workflow capability directly.
- The cache seam now lives once in `internal/infra/cache`; the duplicate cache interfaces and global
  manager were removed, and the rest of `internal/infra/contracts` followed (2026-09) because the
  term contract is reserved for HTTP contracts. Downstream callers inject the byte-oriented
  store instead of selecting mutable process-global drivers.
- Cache request coalescing uses maintained `golang.org/x/sync/singleflight`; the shallow custom
  wrapper package and its cancellation bug were removed.

## Review Rule

Do not add a new exception without also updating:

- this document,
- `.agents/skills/luas-framework-review/scripts/check-api-boundaries.sh`,
- and the rationale in an ADR or roadmap entry.
