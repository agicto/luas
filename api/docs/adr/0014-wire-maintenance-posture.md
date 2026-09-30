# ADR 0014: Wire Maintenance Posture

- Status: Accepted
- Date: 2026-09-30

## Context

The API assembles its dependency graph at compile time with Google Wire. The upstream
`github.com/google/wire` repository was archived on 2025-08-25 and states that it is no longer
maintained. Luas pins `v0.7.0` and runs the generator through `go tool wire`.

A comparison with Kratos, go-zero, uber-go/fx, Goravel, GoFrame, Encore, and Grafana found no reason
to replace compile-time injection. Grafana still uses Wire at a much larger scale. At runtime, Wire
contributes only the small marker package used by `wire.NewSet` and `wire.Bind`. Everything else it
produces is plain Go in `internal/wiring/wire_gen.go`. The measured cost is small: about 1.3 s for a
warm `make wire` and 5.5 s for a warm `go build ./...`.

Before this decision, CI never ran the generator. A provider-set edit without regeneration could
leave a stale `wire_gen.go` that still compiled.

## Decision

1. Keep Wire `v0.7.0` for compile-time assembly. Do not migrate to a runtime container, reflection
   DI, or lazy module construction to reduce graph size.
2. `make wire-check` regenerates the graph and fails when the committed `wire_gen.go` differs. The
   API CI job runs it on every change.
3. `wire_gen.go` is marked `linguist-generated`. Merge conflicts in it are resolved by re-running
   `make wire`, never by hand-merging generated lines.
4. If a supported Go release breaks the pinned generator, take the first option that works:
   switch to a maintained drop-in fork with the same annotations, vendor the generator under a
   Luas-owned module, or, as a last resort, keep the last generated `wire_gen.go` as hand-maintained
   code. Provider sets remain valid documentation in every case.

## Consequences

- Missing or ambiguous dependencies still fail at generation or compile time, which gives coding
  agents a precise error before runtime.
- Upstream abandonment is contained: the runtime dependency surface is only the marker package,
  and the exit paths above do not change module code.
- Revisit this decision when a warm `make wire` exceeds 10 seconds, when Wire breaks on a
  supported Go release, or when downstream applications need providers from outside the repository.
