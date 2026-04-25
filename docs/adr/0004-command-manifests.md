# ADR-0004: Command Manifests

## Status

Accepted

## Context

CLI command registration lived directly in `cmd/zgo/main.go`, while plugin detection maintained a second hard-coded view of the same command surface. This reduced locality: adding or removing a built-in command required editing multiple places, and the command seam could drift away from the actual console application.

## Decision

Introduce `command manifest` as the seam for built-in CLI registration.

- `internal/infra/console/commands` owns built-in command manifests
- `cmd/zgo/main.go` consumes those manifests instead of enumerating commands inline
- plugin detection should ask the console application which commands are registered, instead of maintaining a parallel allowlist

## Consequences

- Built-in CLI registration moves behind one seam.
- Command aliases and canonical names are registered together.
- Future starter or capability specific commands can join the CLI without expanding `cmd/zgo/main.go`.
