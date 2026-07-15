# Permission Starter

The optional `permission` starter is Luas's organization-scoped authorization kernel. Its public
HTTP behavior is defined in [`../../contracts/PERMISSIONS.md`](../../contracts/PERMISSIONS.md).
This document explains the API assembly and downstream extension boundary.

## Activate

The starter has an explicit dependency and fails startup when selected alone:

```dotenv
OPTIONAL_STARTERS=organization,permission
```

Use the same selection for HTTP replicas, migrations, seeders, route inspection, and one-off jobs.
The catalog topologically orders selected manifests, so configuration order is not significant, but
dependencies must be named explicitly.

## Vocabulary And Ownership

- `OrganizationRole` (`owner`, `admin`, `member`) remains owned by `organization` and controls
  ownership, invitations, member removal, and transfer rules.
- `AccessRole` is owned by `permission`, belongs to one organization, and groups exact
  `PermissionKey` values.
- `PermissionContext` is computed for one current membership. It is not embedded in credentials or stored
  as a global current-user object.
- Product resource ownership remains in product policies. An `articles.update` grant answers
  whether the action is available; an article policy still decides whether the caller can update a
  particular article.

## Register Product Permissions

`NewDefaultCatalog` intentionally contains only permission-management keys. A downstream module
extends the catalog at assembly time with reviewed constants:

```go
var (
    PermissionProjectsRead  domain.PermissionKey = "projects.read"
    PermissionProjectsWrite domain.PermissionKey = "projects.write"
)

func NewApplicationPermissionCatalog() (*permission.Catalog, error) {
    keys := append(permission.DefaultPermissionKeys(),
        PermissionProjectsRead,
        PermissionProjectsWrite,
    )
    return permission.NewCatalog(keys...)
}
```

Replace `permission.NewDefaultCatalog` in the application's Wire provider set with the downstream
provider. Duplicate, uppercase, wildcard, one-segment, or overlong keys fail dependency injection
before the server starts. Do not load permission definitions from administrator input or silently
accept unknown persisted grants.

## Service Checks

Inject `domain.PermissionAuthorizer` into business services and pass the typed organization context:

```go
if err := authorizer.Authorize(ctx, organization, PermissionProjectsWrite); err != nil {
    return err
}
```

For HTTP-only checks, inject `*permission.Guard` and attach
`guard.Require(PermissionProjectsWrite)` after `auth` and `organization_context`. Service checks are
still preferred when the same use case can run outside HTTP.

Unknown guard keys return service-unavailable and fail closed. Non-members retain the
organization-not-found privacy branch. Known members without an exact grant return
`PERMISSION.DENIED`.

## Transaction And Revocation Semantics

Read-only authorization uses one indexed membership/assignment/grant join. Mutations run through a
shared repository transaction and lock current membership state plus relevant assignments and
roles before applying delegated-management subset checks. This prevents a manager from using stale
authority to create, modify, remove, or assign permissions above their current effective set.

The starter deliberately does not cache effective permissions across requests. Role updates and
assignment replacement therefore take effect on the next check without a cache-invalidation
protocol. If a downstream app adds a cache, it owns versioning, tenant-scoped keys, revocation
latency, distributed invalidation, and tests proving deny-after-revoke behavior.

## Persistence

The migration owns:

- `permission_roles`
- `permission_role_grants`
- `permission_role_assignments`

Role slugs are unique per organization. Grants are unique per role and permission. Assignments are
unique per organization, membership, and role. Service and repository checks also verify that every
role and membership belongs to the active organization; application checks remain required even
when a database cannot express that cross-table equality as one portable foreign key.

Role and assignment writes emit audit changes after commit. The metadata contains internal IDs and
permission keys, never copied user profiles.

## Replace Or Remove

An external policy engine may replace `domain.PermissionAuthorizer`, but retain the exact public
failure contract and verified organization context. Decide separately whether Luas role-management
routes remain the source of truth or are removed.

To remove the starter, remove `permission` from every `OPTIONAL_STARTERS` value, deploy code without
its routes, then retire its tables through an explicit downstream migration. Also remove the Web
feature, fixed Route Handlers, i18n module, contract references, and product permission catalog.
