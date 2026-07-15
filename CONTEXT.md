# Luas Context

Luas is an AI-era full-stack application scaffold and starter kit. It is not a framework-only kernel, and it is not a finished product application.

This file is the canonical glossary for the whole repository. Use these terms when naming files, packages, features, docs, issues, and agent skills.

## Glossary

**Luas**
: The scaffold itself. Luas provides stable rails for building a downstream application quickly.

**Downstream app**
: An application created from Luas. A downstream app replaces examples, keeps or removes starters, and adds product-specific behavior.

**Scaffold**
: The assembled starting point shipped by this repository: core runtime, default starters, web shell, mock development flows, contracts, docs, and agent guidance.

**Starter kit**
: The product category Luas belongs to. Use this term in public positioning. Avoid describing Luas as only a framework unless talking specifically about reusable runtime internals.

**Core**
: Reusable runtime and infrastructure that every Luas app depends on. Core owns cross-cutting concerns such as bootstrapping, configuration, HTTP plumbing, error handling, logging, testing helpers, routing, and design-system primitives.

**Starter**
: A business-ready building block that ships with the default scaffold. A starter owns a coherent workflow and may include domain rules, persistence, HTTP routes, contracts, mock flows, UI, tests, and docs.

**Default starter**
: A starter wired into the out-of-the-box scaffold. Current default starters are `user`, `apikey`, and `audit`.

**Optional starter**
: A starter-quality building block that exists in the repository but is disabled in the default scaffold. API optional starters are activated additively through the canonical starter catalog; they never replace or subtract the default starter set.

**Capability**
: A reusable technical integration or helper that does not own an application workflow. Examples include AI clients, crypto helpers, ID generation, tracing, storage, queueing, and workflow primitives.

**Feature**
: A user-facing or developer-facing vertical slice of behavior. A feature may have UI, state, services, contracts, tests, and supporting server routes. Feature is the preferred term for product-facing work.

**Module**
: An implementation unit behind an interface or seam. Use module when discussing internal structure, dependency injection, route contribution, or test seams. Do not use module as a synonym for feature unless the seam is the topic.

**Seam**
: A place where behavior can vary without editing callers. Good seams are small, named, testable, and located where the concept naturally belongs.

**Contract**
: Stable HTTP behavior shared across the scaffold. A contract includes request shape, response shape, status code, `error_code`, `request_id`, pagination, and compatibility expectations.

**Mock BFF**
: Development-only route handlers that let the web shell run without a real backend. A mock BFF must preserve the browser-facing contract of the production endpoint or adapter it substitutes, including the shared envelope and error semantics. It is not automatically a mock of every backend endpoint, it is not the production API, and production runtime must require explicit opt-in before serving mock routes.

**Production auth adapter**
: The server-only Web seam that maps the browser auth contract to the Go `user` starter over HTTP. It owns the same-origin HttpOnly access-token cookie, fixed upstream paths and DTO mappings, timeout/error translation, and trusted client-IP forwarding. It is not a generic reverse proxy and never exposes the API bearer token to browser JavaScript.

**Web shell**
: The default browser-facing application surface. It includes route groups, providers, layout, design-system integration, i18n, mock auth, and starter/example UI.

**Console**
: The authenticated scaffold workspace area. Console pages are replaceable starter surfaces, not a finished product dashboard.

**Devtools**
: Internal playground or demonstration routes used to inspect scaffold behavior. Devtools are not product features.

**Example**
: Code or docs whose main purpose is demonstration. Examples must stay isolated and easy for a downstream app to delete.

**Starter registry**
: The assembly point that decides which default and selected optional starters, migrations, seeders, routes, middleware, events, and runtime hooks are active from one configuration snapshot.

**Active organization context**
: The organization membership explicitly selected and revalidated for one API request. It is
  request-scoped, not a global user preference or a long-lived JWT claim. Context-protected routes
  use the canonical organization header and consume the resolved organization ID, membership ID,
  user ID, and role from typed request context rather than trusting raw transport input. The Web
  feature derives browser selection from the current organization URL and forwards it per request;
  it does not persist a global current organization.

**API key scope**
: A lowercase `namespace:action` attenuation attached to one user-owned API key. Scope checks are
  exact unless the explicit `*` wildcard is present. A scope may reduce what the key can do; it does
  not elevate the owning user, represent an organization role, or replace a permission/RBAC model.

**Access role**
: An organization-scoped grouping of exact permission keys owned by the optional `permission`
  starter. Access roles attach to organization memberships. They do not replace the membership
  roles (`owner`, `admin`, `member`) that govern organization lifecycle and ownership.

**Permission key**
: A code-owned lowercase dotted capability such as `projects.read`. Permission checks are exact,
  allow-only, request-scoped to a verified organization membership, and default-deny. Permission
  keys are not API key scopes, runtime administrator input, or resource-ownership policies.

**Production API adapter**
: The Web server-only same-origin boundary that maps explicit browser Route Handlers to fixed API
  operations while owning the HttpOnly API credential, timeout, body budgets, trusted client-IP
  forwarding, response validation, and safe header propagation. It is an allowlist, not an arbitrary
  reverse proxy.

**Command manifest**
: The assembly seam that groups CLI commands so registration does not drift across command packages.

**Default scaffold**
: The out-of-the-box Luas app assembled from core plus the default starter set, web shell, mock BFF, contracts, docs, and verification tooling.

**error_code**
: The canonical machine-readable branch field for non-2xx HTTP responses. Format is uppercase dot-separated scopes such as `COMMON.NOT_FOUND` or `AUTH.INVALID_CREDENTIALS`.

**request_id**
: The correlation identifier carried through logs and error responses when available. Use it to connect a client-visible failure to server-side diagnostics.

## Relationships

- Luas produces a downstream app.
- A scaffold contains core, starters, capabilities, examples, contracts, and agent guidance.
- A starter may span persistence, HTTP routes, contracts, mock BFF behavior, UI, tests, and docs.
- A feature is product-facing behavior; a module is the internal implementation shape behind a seam.
- Core and capabilities are reusable; starters and features express application behavior.
- Contracts connect deployable units. Source code is not shared across deployable units.
- Active organization context is selected per request and verified against current membership.
- API key scopes attenuate a user-owned credential and are not roles or generalized permissions.
- Access roles group exact permission keys inside one active organization; owner bypass and every
  non-owner grant are resolved from current persistence rather than JWT claims.
- The production API adapter connects explicit browser contracts to fixed Go API operations without making unlike contracts pretend to be interchangeable.
- Examples and devtools are disposable. Default starters are out-of-the-box building blocks; optional starters require explicit activation. Core is long-lived infrastructure.

## Flagged Ambiguities

- **framework vs scaffold**: Use scaffold or starter kit for Luas as a whole. Use framework only for reusable runtime internals when that distinction matters.
- **starter vs feature**: Use starter for default or optional Luas-provided building blocks. Use feature for downstream or product-facing slices.
- **module vs feature**: Use module for implementation structure and seams. Use feature for user-facing behavior.
- **mock BFF vs API**: Mock BFF routes mimic contracts for development. The API is the production backend behavior.
- **browser auth contract vs API auth contract**: The Web shell's cookie/session endpoints and the Go API's JWT endpoints are not interchangeable. Use the explicit production API adapter when connecting them; changing `NEXT_PUBLIC_API_URL` alone does not perform the mapping.
- **console vs product dashboard**: Console is a replaceable scaffold workspace. A downstream app may rename or replace it.
- **code vs error_code**: `code` is the transport or success status in the response envelope. `error_code` is the stable machine-readable branch field.
- **API key scope vs role/permission**: API key scopes constrain one credential. Organization roles
  describe membership lifecycle; access roles and dotted permission keys belong to the optional
  `permission` starter. Resource-instance ownership still belongs to the owning business policy.
