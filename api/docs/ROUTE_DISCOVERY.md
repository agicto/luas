# Registered Route Discovery

This document defines how humans, automation, and AI agents discover the HTTP routes assembled by
one Luas API configuration. It separates active topology from HTTP semantics so a list of paths is
not mistaken for a complete API contract.

## Authority

| Surface | Authority |
|---|---|
| [`../../contracts/`](../../contracts/) | Canonical request, response, status, `error_code`, `request_id`, pagination, authentication, and compatibility behavior. |
| `bootstrap.RegisterHTTPRoutes` | One runtime registration seam shared by the HTTP server and route discovery. It owns core operational routes plus active starter routes. |
| `luas route:list` | Human-readable inventory for the currently resolved configuration. |
| `luas route:list --format=json` | Deterministic machine-readable inventory described by [`route-catalog.schema.json`](route-catalog.schema.json). |

The registered route catalog is not an OpenAPI Description. It deliberately contains no inferred
request bodies, response schemas, status codes, authentication rules, middleware claims, or
authorization policy. Those semantics remain in the owning HTTP contract and implementation tests.
Luas currently exposes neither `/swagger` nor `/openapi.json`.

## Commands

From `api/`:

```bash
DB_ENABLED=false AI_ENABLED=false go run ./cmd/luas route:list
DB_ENABLED=false AI_ENABLED=false go run ./cmd/luas route:list --format=json | jq
DB_ENABLED=false AI_ENABLED=false OPTIONAL_STARTERS=organization go run ./cmd/luas route:list --format=json | jq
make route-catalog-check
```

The JSON document is written alone to stdout. Startup diagnostics use stderr, and the CLI suppresses
its interactive completion line in JSON mode, so the command can be piped directly into `jq`, a
JSON Schema validator, an inventory diff, or an agent tool.

Configuration remains authoritative. Development enables metrics by default, while production does
not unless `METRICS_ENABLED=true`; `/metrics` therefore appears only in catalogs whose resolved
configuration registers it. `active_starters` contains the exact registry assembly order, including
default starters and selected optional starters.

## JSON Contract

```json
{
  "kind": "luas.route_catalog",
  "schema_version": 1,
  "active_starters": ["audit", "apikey", "user"],
  "routes": [
    { "method": "GET", "path": "/" },
    { "method": "GET", "path": "/health/ready" },
    { "method": "POST", "path": "/v1/login" }
  ]
}
```

The top-level and route objects are closed. Starter names are canonical lowercase identifiers;
method/path pairs are unique; paths are absolute; and routes are sorted by path then method. The
document omits timestamps, handlers, hostnames, and process IDs so identical assembly produces
stable JSON. A structural change increments `schema_version` and updates the JSON Schema, validator,
tests, docs, and consumers together.

[`../scripts/validate-route-catalog.py`](../scripts/validate-route-catalog.py) provides a dependency-
free strict validator. [`../scripts/verify-route-catalog.sh`](../scripts/verify-route-catalog.sh)
builds the real CLI, runs it from a clean temporary directory so local `.env` files cannot affect
the result, and requires the default starters plus core health, metrics, login, and profile routes.
CI executes that verifier after the API test tier.

## Change Workflow

1. Update the owning file under [`../../contracts/`](../../contracts/) before changing public HTTP behavior.
2. Register the route through core bootstrap or the owning starter manifest, never a parallel documentation registry.
3. Run the relevant handler/contract tests and `make route-catalog-check`.
4. Compare default and affected optional-starter catalogs when activation changes.
5. Update the JSON schema only when the machine output shape changes, not when routes are merely added.

Do not regenerate route truth by parsing Go source with regular expressions. Conditional starter
selection, shared groups, core health routes, metrics policy, and fluent wrappers make source guesses
incomplete. The previous `cmd/tools/routes` and `cmd/tools/apidoc` programs used deleted paths,
omitted runtime-only routes, and have been removed.

## OpenAPI Boundary

The current contract Markdown is explicit but not yet a complete OpenAPI Description. A future OAS
slice must model every included operation's parameters, bodies, success/error envelopes, security,
and optional-starter ownership; validate the description with maintained tooling; and compare path
coverage against this runtime catalog. Until that exists, UI and docs must say `HTTP contracts` or
`registered routes`, never claim that Luas ships OpenAPI or Swagger.

The latest OpenAPI standard is useful precisely because it lets machines understand HTTP semantics,
not merely path names. Publishing a partial generated file would recreate the ambiguity this boundary
removes. See the [OpenAPI Specification](https://spec.openapis.org/oas/latest.html) and the official
[description structure guide](https://learn.openapis.org/specification/structure.html).

## Measured Baseline

Before this boundary, the canonical `route:list` showed 15 application routes and omitted four core
development routes: `/health`, `/health/live`, `/health/ready`, and `/metrics`. The separate legacy
route tool printed a missing-file error but exited successfully, while the documentation generator
failed against the same deleted `routes/v1/register.go` path. Together those tools carried 1,509
lines of inactive parser/generator code.

The shared runtime assembly now reports 19 routes in the default development configuration, 18 in
the production-safe metrics-off configuration, and 32 with the `organization` optional starter in
development. These counts describe the measured configurations, not permanent route budgets.
