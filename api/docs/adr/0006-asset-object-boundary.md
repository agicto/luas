# ADR-0006: Asset Lifecycle And Object-Storage Boundary

## Status

Accepted

## Context

Luas already had filesystem and R2-shaped infrastructure helpers, but it did not have a business
owner for uploaded bytes. A generic storage client cannot decide who owns an upload, which content
is allowed, when bytes become visible, how account deletion behaves, or which identifiers are safe
to expose. Treating a bucket object as the business entity would leak provider semantics into HTTP,
Web, persistence, audit, and downstream products.

The previous local helper also joined caller-controlled paths directly, while the R2 adapter used
AWS SDK for Go v1 after its support window. Those defaults were unsuitable for an enterprise
starter.

## Decision

Luas distinguishes two concepts:

- `asset`: a user-owned business lifecycle record in the optional `asset` starter;
- `stored object`: opaque bytes accessed through the provider-neutral storage capability.

The optional starter owns UUID identity, idempotency, policy, metadata, state transitions,
inspection, account-deletion integrity, audit, cleanup, and HTTP DTOs. It depends on default `user`
and `audit` starters. It exposes neither provider keys nor a generic storage API.

`internal/capabilities/storage` defines small object and direct-transfer interfaces.
`internal/infra/storage` implements disabled, rooted local-development, and R2 adapters. The local
adapter uses rooted filesystem operations, private modes, bounded streams, and atomic writes. The
R2 adapter uses AWS SDK for Go v2, bounded retries/timeouts, and short-lived presigned grants.

Uploads use random staging keys. Completion copies to a distinct immutable final key, then validates
that frozen snapshot's authoritative metadata and bounded content before readiness. Downloads require a fresh
short-lived grant. Signed URLs are credentials and are never durable asset fields.

Production activation requires R2 explicitly. Local storage is an outside-production convenience,
not an implicit production fallback. The browser feature uses fixed management adapters and a
strict ephemeral transfer client; the mock BFF exists only for development parity.

## Consequences

- Downstream modules can depend on `domain.AssetReader` without importing a storage SDK.
- Provider replacement does not change public asset identity or DTOs.
- Readiness is fail-closed and cannot be inferred from a successful upload request alone.
- Account deletion cannot silently orphan active objects.
- Direct upload reduces API bandwidth, but completion still streams one bounded read for content
  integrity; malware scanning and large multipart uploads remain downstream extensions.
- R2 CORS and staging lifecycle rules are deployment responsibilities documented with the starter.
- The optional starter adds no routes, migrations, storage initialization, Web navigation, or bundle
  work until selected.

## Rejected Alternatives

- **Expose filesystem/R2 helpers directly to modules.** This leaves ownership and deletion rules to
  every feature and leaks provider concepts.
- **Use original filenames as keys.** This creates traversal, collision, disclosure, and mutation
  risks.
- **Mark ready after PUT without inspection.** Request metadata is not authoritative and signed PUT
  cannot enforce every provider/body invariant.
- **Proxy every production byte through the API.** This simplifies CORS but imposes avoidable API
  bandwidth and scaling cost; the local token transport remains only a development adapter.
- **Make asset a default starter.** Storage policy and operational cost are product choices, so the
  complete lifecycle remains additive.
