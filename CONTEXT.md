# Luas Context

Luas is an AI-era full-stack application scaffold and starter kit. It is not a framework-only kernel, and it is not a finished product application.

This file is the canonical glossary for the whole repository. Use these terms when naming files, packages, features, docs, issues, and agent skills.

## Glossary

**Luas**
: The scaffold itself. Luas provides stable rails for building a downstream application quickly.

**Downstream app**
: An application created from Luas. A downstream app replaces examples, keeps or removes starters, and adds product-specific behavior.

**Scaffold**
: The assembled starting point shipped by this repository: core runtime, default starters, browser
  shell alternatives, mock development flows, contracts, docs, and agent guidance.

**Starter kit**
: The product category Luas belongs to. Use this term in public positioning. Avoid describing Luas as only a framework unless talking specifically about reusable runtime internals.

**Core**
: Reusable runtime and infrastructure that every Luas app depends on. Core owns cross-cutting concerns such as bootstrapping, configuration, HTTP plumbing, error handling, logging, testing helpers, routing, and design-system primitives.

**Database runtime**
: The core, process-owned GORM and `database/sql` connection boundary. It owns typed connection
  settings, safe DSN construction, bounded pool policy, startup readiness, diagnostics, and
  shutdown. PostgreSQL is the only relational database compatibility authority; SQLite is not a
  valid substitute for migration, repository, or integration evidence. Starters own their schemas
  and query semantics; the database runtime does not turn persistence into a cross-starter business
  module.

**Starter**
: A business-ready building block that ships with the default scaffold. A starter owns a coherent workflow and may include domain rules, persistence, HTTP routes, contracts, mock flows, UI, tests, and docs.

**Default starter**
: A starter wired into the out-of-the-box scaffold. Current default starters are `user`, `apikey`, and `audit`.

**Optional starter**
: A starter-quality building block that exists in the repository but is disabled in the default scaffold. API optional starters are activated additively through the canonical starter catalog; they never replace or subtract the default starter set.

**Capability**
: A reusable technical integration or helper that does not own an application workflow. Examples include AI clients, crypto helpers, ID generation, tracing, storage, queueing, and workflow primitives.

**Cache capability**
: An optional, non-authoritative acceleration seam with explicit byte values, TTL, invalidation,
  adapter, and outage policy. Cached values are disposable copies; cache state is not a business
  record, authentication session, permission authority, idempotency receipt, quota, rate limit,
  durable queue, or distributed lock.

**Feature**
: A user-facing or developer-facing vertical slice of behavior. A feature may have UI, state, services, contracts, tests, and supporting server routes. Feature is the preferred term for product-facing work.

**Module**
: An implementation unit behind an interface or seam. Use module when discussing internal structure, dependency injection, route contribution, or test seams. Do not use module as a synonym for feature unless the seam is the topic.

**Seam**
: A place where behavior can vary without editing callers. Good seams are small, named, testable, and located where the concept naturally belongs.

**Contract**
: Stable HTTP behavior shared across the scaffold. A contract includes request shape, response shape, status code, `error_code`, `request_id`, pagination, and compatibility expectations.

**Mock BFF**
: Development-only Next.js route handlers that let the Web shell run without a real backend. A
  mock BFF must preserve the browser-facing contract of the production endpoint or adapter it
  substitutes, including the shared envelope and error semantics. It is not automatically a mock
  of every backend endpoint, it is not the production API, and production runtime must require
  explicit opt-in before serving mock routes.

**Production auth adapter**
: The server-only Web seam that maps the browser auth contract to the Go `user` starter over HTTP. It owns the same-origin HttpOnly authentication-session cookie, fixed upstream paths and DTO mappings, timeout/error translation, remote logout, and trusted client-IP forwarding. It is not a generic reverse proxy and never exposes the API bearer credential to browser JavaScript.

**Authentication session**
: One signed-in user's server-side, revocable login state. Its bearer credential is opaque and stored only by hash; identity, account status, absolute expiry, idle expiry, and revocation are resolved from current persistence. It is not an API key, authorization claim container, browser-visible token, or global user preference.

**Browser shell**
: A browser-facing application implementation that consumes Luas HTTP contracts. Luas ships the
  Next.js Web shell and the static SPA shell as independent alternatives; downstream apps normally
  select one rather than maintain both.

**Web shell**
: The Next.js browser-shell implementation under `web/`. It includes route groups, providers,
  layout, design-system integration, i18n, mock auth, server-only production adapters, and
  starter/example UI.

**Static SPA shell**
: The Vite and TanStack Router browser-shell implementation under `web-spa/`. It preserves the
  feature-first, contract, error, state, i18n, and design-system architecture while emitting only
  static OSS/CDN assets. It has no BFF, server function, secret environment, or production Node.js
  runtime.

**Browser gateway**
: A same-origin server or ingress boundary used by a static browser shell for fixed API operations,
  HttpOnly credential custody, Origin/CSRF enforcement, and bounded upstream mapping. It is not an
  arbitrary reverse proxy and is not implemented by static JavaScript.

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
  request-scoped, not a global user preference or an authentication-session claim. Context-protected routes
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

**Notification**
: One immutable application event addressed to one user by the optional `notification` starter.
  It owns user-visible plain-text content, a caller-owned idempotency key, optional relative action
  URL, and mutable read state only when an `in_app` delivery exists. It is not a provider message.

**Notification delivery**
: One channel-specific execution record for a notification. A delivery owns channel, state,
  attempts, availability, worker lease, stable failure code, and completion time without storing
  provider response bodies or recipient addresses. `in_app` and `email` are the starter's current
  channels; provider-specific payloads remain adapter details.

**Asset**
: A user-owned metadata record and lifecycle for one opaque stored object. The optional `asset`
  starter owns upload intent, validation, readiness, private download, and deletion policy. An asset
  is not a filesystem path, browser `File`, public URL, or storage-provider object by itself.

**Stored object**
: The opaque bytes addressed by an application-generated key behind the object-storage capability.
  Staging and final objects are provider details and never become owner or asset identifiers in HTTP
  contracts. Feature code depends on the capability seam rather than an R2/S3 SDK.

**Setting definition**
: A code-owned schema for one bounded app, organization, or user preference or policy value. It
  owns the semantic key, scalar kind, visibility, default, options, and validation. Definitions are
  reviewed code; they are not administrator-created key/value rows.

**Setting override**
: A durable value selected for one setting definition and one scope subject. Reset keeps version
  history while returning to the code default. An override is not process configuration, a secret,
  a permission grant, an entitlement, or a notification channel preference.

**Effective setting**
: The value returned by combining a setting definition with its current override or code default.
  It carries source and monotonic version metadata so clients can cache and mutate it safely.

**Usage metric**
: A code-owned additive measure such as `api.requests` or `ai.input_tokens`. A metric owns its
  semantic key, unit, UTC aggregation period, optional default hard limit, and finite dimension
  schema. It is not an observability metric name, price, invoice item, or runtime-created meter.

**Usage event**
: One immutable, internally reported usage fact or correction for a metric and a user or
  organization subject. The pair of source and event ID owns retry idempotency. Arbitrary browser
  ingestion, payload JSON, identifiers in dimensions, and provider billing events are outside this
  meaning.

**Usage counter**
: The durable non-negative aggregate for one usage metric, subject, and UTC calendar period. A
  counter is updated atomically with its receipt and survives receipt pruning; it is not an HTTP
  rate-limit bucket or an analytics time series.

**Usage quota**
: The optional hard integer cap resolved from a metric default and a monotonic subject override.
  Atomic consumption may reject work that would cross the cap. A quota is not a plan, grant,
  entitlement, balance, price, or payment-provider object.

**Webhook endpoint**
: An organization-owned outbound HTTPS target with a finite event subscription, active/disabled
  state, monotonic version, and endpoint-unique encrypted signing secret. It is not an inbound
  callback, arbitrary proxy destination, provider credential, or browser-owned request template.

**Webhook event**
: One trusted code-owned outbound publication with a finite type, producer idempotency identity,
  stable `msg_` identifier, occurrence time, and validated payload. Browser clients cannot publish
  arbitrary webhook events; downstream server modules use the durable publisher seam.

**Webhook delivery**
: The durable execution ledger entry for one webhook event and endpoint. It owns retry state,
  lease, attempts, stable local failure identifiers, and replay count without exposing target URLs,
  payloads, signatures, response bodies, or free-form provider errors.

**Production API adapter**
: The Web server-only same-origin boundary that maps explicit browser Route Handlers to fixed API
  operations while owning the HttpOnly API credential, timeout, body budgets, trusted client-IP
  forwarding, response validation, and safe header propagation. It is an allowlist, not an arbitrary
  reverse proxy.

**Command manifest**
: The assembly seam that groups CLI commands so registration does not drift across command packages.

**Default scaffold**
: The out-of-the-box Luas source assembled from core plus the default starter set, browser-shell
  alternatives, contracts, docs, and verification tooling. A downstream app selects the Next.js
  Web shell or static SPA shell and removes the alternative it will not maintain.

**error_code**
: The canonical machine-readable branch field for non-2xx HTTP responses. Format is uppercase dot-separated scopes such as `COMMON.NOT_FOUND` or `AUTH.INVALID_CREDENTIALS`.

**request_id**
: The correlation identifier carried through logs and error responses when available. Use it to connect a client-visible failure to server-side diagnostics.

**Performance budget**
: A reviewed, executable upper bound on one repeatable build or runtime measurement. A budget owns
  its metric, scope, evidence source, and change procedure. It is a regression guard, not capacity
  to consume, a field Core Web Vital, or a service-level objective.

**Field Web Vital**
: A real-user LCP, INP, or CLS measurement from production traffic, evaluated at the 75th
  percentile under a declared device and reporting policy. Synthetic Lighthouse and build bundle
  evidence help diagnose performance but do not establish a field Web Vital.

## Relationships

- Luas produces a downstream app.
- A scaffold contains core, starters, capabilities, examples, contracts, and agent guidance.
- The database runtime owns connections and pool lifecycle; each starter keeps schema, transaction,
  query, ordering, and pagination behavior local to its repository.
- The cache capability may accelerate an owning starter or downstream feature, but the caller keeps
  authoritative reads, invalidation, staleness, tenant key scope, and failure behavior local.
- A starter may span persistence, HTTP routes, contracts, mock BFF behavior, UI, tests, and docs.
- A feature is product-facing behavior; a module is the internal implementation shape behind a seam.
- Core and capabilities are reusable; starters and features express application behavior.
- Contracts connect deployable units. Source code is not shared across the API, Web shell, or
  static SPA shell.
- The Web shell and static SPA shell are alternative browser implementations. Behavioral parity
  comes from contracts and tests, not source imports.
- The static SPA shell requires a browser gateway or explicit browser-session API before protected
  authentication is production-complete; browser storage never replaces HttpOnly credential
  custody.
- Active organization context is selected per request and verified against current membership.
- API key scopes attenuate a user-owned credential and are not roles or generalized permissions.
- Authentication sessions identify current signed-in users; API keys identify machine/API access.
  Neither credential carries current organization roles or permission grants as trusted claims.
- Access roles group exact permission keys inside one active organization; owner bypass and every
  non-owner grant are resolved from current persistence rather than credential claims.
- A notification represents the user-facing event once; notification deliveries represent its
  independently executed channels and retry lifecycle.
- An asset owns business metadata and lifecycle; a stored object owns provider-neutral byte
  operations. Signed transfer URLs are short-lived bearer credentials, not durable asset identity.
- A setting definition owns type/default/visibility in code; a setting override owns one durable
  scoped choice; an effective setting exposes the resolved value and version.
- A usage metric defines what can be counted; a usage event records one idempotent fact; a usage
  counter aggregates one subject period; a usage quota optionally gates atomic consumption.
- A webhook event represents one trusted outbound fact; each subscribed webhook endpoint receives
  a separate durable delivery whose message identity survives retry and replay.
- The production API adapter connects explicit browser contracts to fixed Go API operations without making unlike contracts pretend to be interchangeable.
- A performance budget guards a stable engineering signal; field Web Vitals describe production
  user outcomes. Neither term substitutes for the other.
- Examples and devtools are disposable. Default starters are out-of-the-box building blocks; optional starters require explicit activation. Core is long-lived infrastructure.

## Flagged Ambiguities

- **framework vs scaffold**: Use scaffold or starter kit for Luas as a whole. Use framework only for reusable runtime internals when that distinction matters.
- **starter vs feature**: Use starter for default or optional Luas-provided building blocks. Use feature for downstream or product-facing slices.
- **module vs feature**: Use module for implementation structure and seams. Use feature for user-facing behavior.
- **mock BFF vs API**: Mock BFF routes mimic contracts for development. The API is the production backend behavior.
- **browser auth contract vs API auth contract**: The Web shell's cookie/session endpoints and the Go API's opaque authentication-session endpoints are not interchangeable. Use the explicit production auth adapter when connecting them; changing `NEXT_PUBLIC_API_URL` alone does not perform the mapping.
- **Web shell vs static SPA shell**: Web means the Next.js implementation with server-owned
  adapters and mock BFF routes. Static SPA means the Vite/TanStack implementation deployed as
  browser assets. They share contracts and vocabulary, not source or server capabilities.
- **static SPA vs browser gateway**: The SPA owns browser UI and public build configuration. A
  browser gateway owns HttpOnly credentials, unsafe-request protection, and fixed server mappings;
  CDN fallback routing or browser storage is not a gateway.
- **console vs product dashboard**: Console is a replaceable scaffold workspace. A downstream app may rename or replace it.
- **code vs error_code**: `code` is the transport or success status in the response envelope. `error_code` is the stable machine-readable branch field.
- **API key scope vs role/permission**: API key scopes constrain one credential. Organization roles
  describe membership lifecycle; access roles and dotted permission keys belong to the optional
  `permission` starter. Resource-instance ownership still belongs to the owning business policy.
- **notification vs delivery/message**: A notification is the immutable user event. A notification
  delivery is one channel execution ledger entry. Provider messages, receipts, and callback payloads
  are adapter details and must not create a second global meaning for notification.
- **asset vs file/object**: Use asset for the user-owned workflow record. Use stored object for bytes
  behind the object-storage capability. Use file only for a local operating-system or browser value;
  never expose a provider key or filesystem path as the asset identifier.
- **setting vs config/secret/policy**: Use setting for a code-defined durable preference or policy
  override. Process configuration is restart-scoped typed infrastructure authority; secrets stay in
  environment/provider stores; permissions, entitlements, usage limits, and notification channel
  preferences remain with their owning starters.
- **usage vs telemetry/rate limit/billing**: Usage is durable business accounting for a code-owned
  metric and subject. Telemetry explains system operation, HTTP rate limits protect transport,
  quotas gate a usage counter, and billing converts reviewed usage into provider-specific money.
  Do not use these terms interchangeably.
- **cache vs state/coordination**: Cache means disposable acceleration data. Durable state,
  authentication sessions, idempotency, quotas, rate limits, queues, and distributed locks retain
  their own authorities and atomicity contracts; a generic cache operation does not replace them.
- **webhook vs event bus/workflow/inbound callback**: Webhook means an outbound, signed HTTP
  delivery owned by the optional starter. The in-process event bus is best-effort process
  coordination, workflow owns generic execution primitives, and inbound provider callbacks require
  a separate authenticated contract. Do not use webhook as a synonym for any of them.
