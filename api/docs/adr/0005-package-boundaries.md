# ADR-0005: Package Boundaries — `pkg/`, `internal/domain/`, `internal/capabilities/`, `internal/infra/`

## Status

Accepted

## Context

Luas has four API homes that can plausibly host shared code or shared vocabulary:

- `pkg/` — currently houses `encryption`, `env`, `errors`, `events`, `handler`,
  `hash`, `logger`, `pagination`, `request`, `resource`, `response`, `support`,
  `utils`, `validation`.
- `internal/domain/` — currently houses starter domain entities, domain errors,
  domain events, value objects, repository interfaces, and stable API `error_code`
  constants.
- `internal/capabilities/` — currently houses `ai`, `crypto`, `idgen`.
- `internal/infra/` — currently houses `config`, `console`, `events`, `exception`,
  `http`, `lang`, `logger`, `middleware`, `migration`, `plugin`, `provider`,
  `ratelimit`, `routes`, `storage`, `telemetry`, `tracing`, `validation`.

New contributors (and AI agents) routinely ask "which one do I put it in?"
Without a written rule, the boundary drifts. There were already two crypto
packages (`pkg/encryption` and `internal/capabilities/crypto`) before the
public-release cleanup.

## Decision

We treat these homes as sharply different concerns, and we
write them down so reviewers and AI agents can apply the rule
mechanically.

### `pkg/` — reusable, framework-agnostic utilities

A package belongs in `pkg/` when **all** of the following hold:

1. It has **no dependency** on `internal/...`, on `gin`, on `gorm`, or on
   any other framework-level choice this scaffold makes.
2. It is a stable building block that other Go projects could vendor in
   without inheriting Luas's architecture.
3. It owns no global state, no init-time side effects, no DI wiring.

Examples that fit: `pkg/env`, `pkg/pagination`, `pkg/errors`.

Smell test: would a Go developer outside Luas reasonably copy this file
into their own project unchanged? If yes, `pkg/`.

### `internal/domain/` — framework-free starter vocabulary

A file belongs in `internal/domain/` when:

1. It names a domain concept used by one or more starter modules.
2. It can run with only the Go standard library.
3. It defines a small seam such as a repository interface, domain error,
   value object, event, or `error_code` constant without importing HTTP,
   database, runtime, or response helpers.

Examples that fit: `domain.User`, `domain.APIKey`, `domain.AuditLog`,
`domain.UserRepository`, `domain.CodeInvalidCredentials`.

Smell test: would importing this from `internal/infra/` make the runtime
know about starter behavior? If yes, keep the dependency out of infra and
let modules or assembly code adapt the domain concept instead.

### `internal/capabilities/` — opinionated, domain-shaped capabilities

A package belongs in `internal/capabilities/` when:

1. It exposes a **provider-neutral surface** the rest of the app calls
   through (e.g. `ai.Provider`, `idgen.Generator`).
2. The surface is **Luas-shaped**: it makes sense in *this* application's
   world but may not generalize. (`capabilities/ai` is what Luas's
   modules and CLI need, not the OpenAI SDK reshape.)
3. It can be **Wire-injected** into modules without leaking framework
   details (no `*gin.Context`, no `*gorm.DB` in its public API).

Examples that fit: `capabilities/ai`, `capabilities/idgen`.

Smell test: would another Luas-based app reuse this verbatim, but a
plain Go service probably wouldn't bother? If yes, `internal/capabilities/`.

### `internal/infra/` — framework wiring + runtime glue

A package belongs in `internal/infra/` when:

1. It is **bound to specific framework / runtime decisions** Luas has
   made (Gin, GORM, OTel, our env-loader, our console framework).
2. It exists to make the runtime work, not to be reused. Replacing the
   framework would replace this code.
3. It frequently imports from `pkg/` and from `internal/capabilities/`,
   but **the reverse is forbidden**.

Examples that fit: `infra/config`, `infra/middleware`, `infra/migration`,
`infra/tracing`, `infra/http` (the JSON HTTP client wrapper).

Smell test: if we swapped Gin for Echo, would this file rewrite? If yes,
`internal/infra/`.

## Dependency rule

The allowed import direction is strictly one-way:

```
internal/domain/  <--  internal/modules/

pkg/  <--  internal/capabilities/  <--  internal/infra/  <--  internal/modules/
                                                        ^
                                                        |
                                              cmd/, internal/bootstrap/
```

Concretely:

- `pkg/` may **not** import `internal/...`.
- `internal/domain/` may **not** import `pkg/` or `internal/...`.
- `internal/capabilities/` may import `pkg/` but **not** `internal/domain/`,
  `internal/infra/`, or `internal/modules/`.
- `internal/infra/` may import `pkg/` and `internal/capabilities/`, but
  not `internal/domain/` or `internal/modules/`.
- `internal/modules/` may import `internal/domain/` and the lower runtime
  layers; modules are the leaf of the graph.

If a piece of code wants to import "up" the chain, that's the signal it
lives at the wrong layer — promote/demote it instead of adding the
import.

## Consequences

- New PRs that add code to `pkg/` that imports `internal/...` are
  flagged in review as boundary violations.
- New PRs that make `internal/domain/` depend on runtime helpers are
  flagged before domain concepts become framework-coupled.
- Luas-branded startup output belongs in `internal/bootstrap/`, not in
  reusable `pkg/` helpers.
- The previous duplicate (`pkg/encryption` ↔ `internal/capabilities/crypto`)
  was resolved by keeping the capability-shaped `crypto` and treating
  `pkg/encryption` as deprecated.
- `make:module` and the AGENTS.md template tell new contributors to
  always place new code under `internal/modules/<name>/` unless it
  truly belongs in one of these documented homes — and to point
  to this ADR when justifying the choice.
- The boundary lint lives at
  `.agents/skills/luas-framework-review/scripts/check-api-boundaries.sh`
  from the repository root. It blocks new reverse imports while any current
  baseline exceptions, if they exist, are tracked in
  [`../PACKAGE_BOUNDARIES.md`](../PACKAGE_BOUNDARIES.md).
