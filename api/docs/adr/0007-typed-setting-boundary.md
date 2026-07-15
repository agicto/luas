# ADR 0007: Typed Setting Boundary

- Status: accepted
- Date: 2026-07-15

## Context

Luas had a replaceable Web settings page but no durable API-owned setting behavior. Repeated
downstream projects need branding, locale defaults, user preferences, and organization policy, yet
an unrestricted key/value table would erase schema ownership, permit secret misuse, weaken cache
invalidation, and make clients branch on undocumented data.

Process configuration already has one typed startup authority. Permission, notification, and asset
starters already own their respective policy, preference, and resource lifecycles. A new boundary
must not absorb those domains.

## Decision

`setting` is an organization-dependent optional starter with a code-owned catalog and durable
overrides at `app`, `organization`, and `user` scope.

- Definitions own key, scope, scalar kind, visibility, default, options, and validation.
- Persistence stores only overrides or reset tombstones. Code remains authoritative for defaults.
- The catalog is finite and bounded; HTTP cannot create definitions or arbitrary JSON values.
- Every mutation uses a strong version precondition and compare-and-swap semantics.
- Reset increments a retained version so stale writers cannot pass after an override disappears.
- Public app values use aggregate ETag caching. Every private response is non-cacheable.
- Organization membership roles own organization mutation authorization; permission roles do not
  silently change this lifecycle invariant.
- Audit and logs record metadata and versions only, never setting values.
- Operator set/reset commands persist a minimized system-actor audit entry after the setting write;
  audit failure is warned and does not roll back an already committed override.
- User setting cleanup participates in the account-deletion transaction after every guard passes.
- App-scope mutation is operator/internal behavior in v1; it is not exposed as a public admin API.

The shipped semantic keys are `branding.display_name`, `localization.locale`, and
`localization.timezone`. The same key may exist at more than one scope when its meaning is identical.

## Consequences

Downstream code gets stable typed settings without paying for a remote flag platform. Clients can
detect concurrent edits, caches can revalidate exactly, and default changes remain reviewed code
changes. Adding a definition requires API and Web schema/UI work when browser exposure is intended.

The starter deliberately cannot store arbitrary product documents or secrets. Dynamic rollout,
entitlements, and hot process configuration require separate systems. Public organization values
remain unavailable until Luas defines a safe public organization identity contract.

## Alternatives Rejected

- **One JSON document per tenant:** weak field-level validation and concurrency; unbounded rewrite
  cost and merge ambiguity.
- **Arbitrary key/value rows:** no trustworthy catalog, visibility, or downstream compatibility.
- **Environment variables only:** no user/organization ownership or durable preference workflow.
- **Permission-backed setting writes:** makes a simple lifecycle starter depend on optional RBAC and
  changes owner/admin semantics when that starter is toggled.
- **Delete rows on reset:** permits ABA races because an old expected version can become valid again.
