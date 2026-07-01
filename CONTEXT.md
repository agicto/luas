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
: A starter-quality building block that exists in the repository but is not wired into the default scaffold.

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
: Development-only route handlers that let the web shell run without a real backend. A mock BFF must emit the same contract shape as the real API. It is not the production API, and production runtime must require explicit opt-in before serving mock routes.

**Web shell**
: The default browser-facing application surface. It includes route groups, providers, layout, design-system integration, i18n, mock auth, and starter/example UI.

**Console**
: The authenticated scaffold workspace area. Console pages are replaceable starter surfaces, not a finished product dashboard.

**Devtools**
: Internal playground or demonstration routes used to inspect scaffold behavior. Devtools are not product features.

**Example**
: Code or docs whose main purpose is demonstration. Examples must stay isolated and easy for a downstream app to delete.

**Starter registry**
: The assembly point that decides which starters, migrations, seeders, routes, middleware, events, and supporting surfaces are active in the default scaffold.

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
- Examples and devtools are disposable. Starters are default building blocks. Core is long-lived infrastructure.

## Flagged Ambiguities

- **framework vs scaffold**: Use scaffold or starter kit for Luas as a whole. Use framework only for reusable runtime internals when that distinction matters.
- **starter vs feature**: Use starter for default or optional Luas-provided building blocks. Use feature for downstream or product-facing slices.
- **module vs feature**: Use module for implementation structure and seams. Use feature for user-facing behavior.
- **mock BFF vs API**: Mock BFF routes mimic contracts for development. The API is the production backend behavior.
- **console vs product dashboard**: Console is a replaceable scaffold workspace. A downstream app may rename or replace it.
- **code vs error_code**: `code` is the transport or success status in the response envelope. `error_code` is the stable machine-readable branch field.
