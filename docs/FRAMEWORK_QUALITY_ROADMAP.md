# Luas Framework Quality Roadmap

This roadmap tracks long-running work to keep Luas professional, semantically clear, architecture-friendly, and AI-agent friendly.

Use [`../CONTEXT.md`](../CONTEXT.md) for vocabulary. Use the `luas-framework-review` skill before adding or re-ranking items.
Use [`SKILL_GOVERNANCE_PLAN.md`](SKILL_GOVERNANCE_PLAN.md) for the 30/60/90-day plan that keeps agent skills aligned with Luas vocabulary, contracts, and architecture.

## Quality Axes

| Axis | Target |
|---|---|
| Semantic clarity | Names, docs, routes, contracts, and skills use the global vocabulary consistently. |
| Architecture depth | Important behavior sits behind small, named seams with clear ownership and test surfaces. |
| Contract integrity | API and mock BFF behavior share status codes, response envelopes, `error_code`, `request_id`, and pagination rules. |
| Security defaults | Production defaults are safe without surprising configuration gaps. |
| Performance baseline | Performance claims are backed by build output, bundle evidence, timings, or benchmarks. |
| Usability | A downstream app can delete examples, keep starters, and add features without hunting through unrelated layers. |
| AI workflows | Agents can find context, choose skills, make scoped changes, and verify outcomes without re-learning the repo each turn. |

## Current Baseline

- Global vocabulary now lives in [`../CONTEXT.md`](../CONTEXT.md).
- API and Web remain independent deployable units and share contracts, not source code.
- Error contracts have been aligned around `code`, `error_code`, `message`, optional `errors`, and optional `request_id`.
- Scaffold-level error contracts are guarded by `.agents/skills/luas-framework-review/scripts/check-error-contracts.py`, keeping `contracts/README.md`, API response constants, and Web status fallbacks aligned.
- API default HTTP guardrails now include security headers, request body limit, cooperative request timeout, production-default rate limiting, CORS, and standard `error_code` responses for body-limit, timeout, and rate-limit failures.
- API transport configuration now owns the real socket boundary: local defaults bind to `127.0.0.1`,
  container surfaces explicitly bind `0.0.0.0`, and read-header/read/write/idle/header-size budgets
  are wired into `http.Server`. Configuration validation rejects negative values and a positive write
  deadline that cannot outlive the cooperative request timeout.
- API configuration now has one typed startup authority with deterministic environment-file
  precedence. Runtime-only process values remain highest priority, stale file values are removed by
  test/diagnostic reloads, and misleading dynamic-key, hot-reload, and no-op cache surfaces have been
  removed. The lifecycle and secret ownership contract lives in
  [`../api/docs/CONFIGURATION.md`](../api/docs/CONFIGURATION.md).
- Outbound email now uses one reusable HTTP client, caller context plus a 10-second provider budget,
  a 50-recipient request cap, 64 KiB response cap, all-or-none typed configuration, address
  validation, HTML-escaped template values, and status-only provider errors. Recipient, subject,
  body, credential, and provider-body
  data stay out of logs and returned errors; direct delivery remains explicitly best-effort rather
  than pretending to be a notification workflow. Against `e6ff2a1`, a Go 1.25.12 `darwin/arm64`
  stripped `cmd/server` moved from 34,544,338 to 34,527,826 bytes (-16,512, -0.048%), while the
  `go list -deps` package count stayed at 632. These are binary/dependency measurements, not an email
  provider latency claim. See [`../api/docs/EMAIL.md`](../api/docs/EMAIL.md).
- The API production image no longer embeds development environment files. It runs non-root with
  production/release defaults, JSON request logs on stdout, file logging disabled, and an executable
  loopback liveness check. Local Compose is explicitly development-only, and CI builds and exercises
  the same image contract through `make container-check`.
- API client-IP controls now deny forwarding-header trust by default, validate exact `SERVER_TRUSTED_PROXIES`, and reject trust-all networks. Public auth routes add production-default independent per-IP/per-subject quotas, hashed subject keys, generic `COMMON.RATE_LIMITED` responses without bucket diagnostics, one-query login lookup, fixed dummy-hash work for unknown accounts, and the same `AUTH.INVALID_CREDENTIALS` response for unknown, wrong-password, and disabled accounts.
- The API minimum toolchain is now Go 1.25.12 and `quic-go` is at 0.59.1, closing the reachable standard-library and HTTP/3 findings reported against Go 1.25.0 / `quic-go` 0.58.0. A full `govulncheck ./...` reports zero reachable vulnerabilities; three advisories remain only in required modules with no called symbols. With both trees built by Go 1.25.12 using `-trimpath -ldflags='-s -w'`, the auth/proxy/tooling slice moves `cmd/server` from 34,412,002 to 34,445,090 bytes (+33,088 bytes, 0.10%) against baseline `fcb58b1`; `x/tools` and `x/vuln` remain absent from the server package dependency graph.
- API operational routes now keep health probes always available while Prometheus instrumentation and `/metrics` follow `METRICS_ENABLED` (enabled outside production, disabled by default in production). Unmatched URLs collapse to one bounded metric label, and the broken default `/monitor` and `/swagger` surfaces have been removed until they have real assembly and contracts.
- Removing the unwired Swagger runtime dependencies reduced the local Go module graph from 298 to 271 modules and the stripped `cmd/server` binary from 44,835,362 to 34,395,426 bytes (23.29%) on Go 1.25.0 `darwin/arm64`. This is a dependency and binary-footprint baseline measured with `go list -m all` and `go build -trimpath -ldflags='-s -w'`; it is not a throughput claim.
- Compression is intentionally not part of the default API kernel; prefer deployment/CDN compression or explicit route/starter middleware.
- API middleware ownership is now cataloged in [`../api/docs/MIDDLEWARE.md`](../api/docs/MIDDLEWARE.md).
- Web error-code vocabulary is contract-tested so `ApiErrorCode` remains server-scoped, `ClientErrorCode` remains frontend-only, and legacy underscore codes stay normalization input only.
- Web mock BFF routes are disabled in production runtime by default through `guardMockBffRoute()` and require explicit `MOCK_BFF_ENABLED=true` opt-in for demo-only deployments.
- Authenticated Web Route Handlers now declare their intermediary-cache boundary explicitly. Every
  auth success/failure and organization response uses `Cache-Control: private, no-store`; auth and
  organization routes vary on `Cookie`, while organization context also retains `Organization-Id`.
  A shared server-only helper merges existing `Vary`, request-ID, and rate-limit evidence, and route
  contract tests prevent new auth or organization handlers from bypassing the finalizer.
- Web route handlers are contract-tested so mock-only routes call `guardMockBffRoute()`, hybrid auth
  routes call `resolveAuthRoute()`, unsafe mutations apply the same-origin guard after availability,
  success envelopes use shared helpers, and legacy underscore-style error codes stay absent.
- The default API key starter now has atomic idempotent revocation that cannot be cleared by a stale
  usage write, throttled `last_used_at`, structured JSON scope storage with legacy read compatibility,
  bounded `namespace:action` scope grammar, and a route-level exact scope guard. The Web replaces its
  fabricated settings key with strict fixed-path production/mock routes and a real create/list/revoke
  workflow; plaintext is shown once, immediately removed from mutation state, and forbidden from list
  metadata. [`../contracts/API_KEYS.md`](../contracts/API_KEYS.md) and an executable boundary check own
  the cross-service semantics.
- Sensitive telemetry now has one standard-library-only `pkg/redact` boundary. HTTP logs use route
  templates (plus a bounded unmatched sentinel) and omit concrete paths/query values/bodies; HTTP
  traces omit concrete paths and free-form errors; logger context and audit metadata are recursively
  sanitized; the development exception center redacts credential headers/query values and escapes
  every dynamic HTML field; and all GORM trace paths are permanently parameterized. An executable
  governance check keeps these seams aligned with [`../api/docs/OBSERVABILITY.md`](../api/docs/OBSERVABILITY.md).
- Web mock BFF replacement is documented in [`../web/docs/MOCK_BFF.md`](../web/docs/MOCK_BFF.md), including production modes, deletion seams, and verification.
- Web Query/Auth providers are route-scoped: root keeps only app-wide UI context, `(auth)` owns React Query mutations, and `(protected)` owns authenticated providers.
- Web public route hydration boundaries are guarded by `src/test/public-route-boundary.test.ts`, which blocks auth, query, HTTP, mock BFF, mock session, toast, and Zustand runtime dependencies from `(site)` routes.
- The auth visual shell is now a Server Component, while `LanguageSwitcher`, forms, and `QueryProvider` remain client leaves. Moving Zod validation behind the server-only environment boundary also removed the full validator from browser chunks. On Next.js 16.2.9, the `/login` route client entry set fell from 702,002 to 409,986 raw bytes and from 195,929 to 129,095 gzip bytes (41.60% and 34.11%), and the auth layout itself left the client reference graph. This is build-manifest evidence, not a field Core Web Vitals claim.
- Root runtime ownership now uses Next.js `error.tsx` / `global-error.tsx`, keeps optional analytics as a Server Component, and scopes Sonner to `(auth)` / `(protected)`. The root client entry fell from 179,070 to 106,547 raw bytes and from 51,479 to 29,335 gzip bytes (40.50% and 43.02%); the public site route entry fell from 271,306 to 232,813 raw bytes and from 82,784 to 71,914 gzip bytes (14.19% and 13.13%). These are build-manifest measurements, not field Core Web Vitals.
- Client-side i18n messages are now namespace-scoped: root serializes only `common` / `errors`, while auth, console, and the i18n devtool append their owned namespaces. In production HTML sampling, `/` fell from 69,703 to 64,642 raw bytes and from 14,758 to 12,264 gzip bytes (7.26% and 16.90%); `/login` fell from 42,455 to 40,765 raw bytes and from 9,486 to 8,695 gzip bytes (3.98% and 8.34%). The login client entry increased by 407 raw / 124 gzip bytes for the additive route provider, leaving a net first-load reduction of 1,283 raw / 667 gzip bytes. These are local production-build transfer measurements, not field Core Web Vitals.
- Web i18n exposes `useT` through the client-safe `@/i18n` entry and `getT` through `@/i18n/server`; `src/test/i18n-runtime-boundary.test.ts` prevents server imports or the full auth shell from leaking back into the client graph.
- Web i18n scope semantics are now message-tree-derived: the current tree produces 36 valid object scopes and 232 translatable leaf keys, while `ScopedTranslations<P>` accepts only relative leaf keys below `P`. `src/test/i18n-types.test.ts` guards the compile-time contract and runtime prefix composition. The type-system refactor itself kept production manifests byte-identical for root, site, login, and console client entries, with no i18n loader, server entry, or message source added to the browser graph.
- Web i18n interpolation semantics are now base-locale-derived: the current tree has 12 variable-bearing messages, and global, namespace, and scoped translators require their exact ICU variables while rejecting values for static messages. Configured locale coverage and ICU variable-name parity are compile-time contracts, so translated `{name}` / `{year}` / `{value}` drift fails before runtime. Isolated production builds against baseline commit `402417f` produced byte-for-byte identical static JavaScript: 33 chunks, 1,378,334 bytes, and content-set SHA-256 `ca19cdbe78e87377945624466114b5540892409ccfe46aa653ac50203b9734b1`.
- Core Web copy now has an executable surface boundary: `scripts/check-i18n-copy.mjs` reduced the initial AST baseline from 98 hardcoded user-facing literals to zero across 21 formal site, auth, console, root metadata, and shared-shell files, with 7 exact brand literals explicitly allowed. `pnpm lint` runs the guard; `devtools`, `example`, technical `<code>` content, and the dependency-light root fallback remain deliberately outside it. Translation ownership now uses `site`, `console`, and `settings` instead of the stale product-like `dashboard` message namespace.
- Shared Web form primitives now preserve native `Input` semantics instead of silently replacing date/color types, share stable error ids and merged `aria-describedby` behavior, expose polite error announcements, require caller-owned labels for password visibility actions, and keep the visual color picker backed by a focusable native color input. `src/test/form-control-accessibility.test.tsx` guards these public contracts. Removing the implicit DatePicker dependency reduced isolated production-build `/login` and `/register` client entry sets from 13 to 11 chunks, 412,501 to 397,825 raw bytes (3.56%), and 130,529 to 125,914 gzip bytes (3.54%) against baseline commit `e15e83b`; these are build-manifest measurements, not field Core Web Vitals.
- Calendar and DatePicker now delegate locale week rules, grid semantics, focus movement, and keyboard navigation to React DayPicker 10 instead of a hand-built date grid. Configured locales have an exhaustive calendar-locale registry; DatePicker exposes a labeled combobox/dialog contract, merged form errors, browser-local hidden values, Intl display formatting, and native time selects instead of 144 time buttons. The engine is interaction-loaded: the `/styleguide` initial client set stayed at 13 chunks and moved from 517,845 to 516,722 raw bytes, while the calendar is a 78,866 raw / 23,586 gzip byte async chunk; `/login` and `/register` remain at 11 chunks and 397,872 raw bytes. These are local production-manifest measurements, not field Core Web Vitals.
- The public auth shell now exposes one `main` landmark, guarded by `src/test/route-accessibility-contract.test.ts`. Local production Lighthouse accessibility moved from 98 to 100 on `/login`, while `/register` also scored 100 with zero failed accessibility audits; this is a synthetic automated audit, not a substitute for assistive-technology testing.
- Web theme and shared-primitive accessibility now have executable contracts: `pnpm lint:theme-contrast` parses the actual OKLCH token graph and guards 48 light/dark semantic text pairs at WCAG AA 4.5:1; `AvatarImage` requires explicit alt semantics; `AlertTitle` no longer injects a fixed heading level; icon-only loading examples are named; and both the styleguide page and protected loading state expose one `main` landmark. Mobile Lighthouse on `/styleguide` moved from 85 with five failed audit families to 100 with zero failures in forced light and dark modes. The production client entry stayed effectively flat at 13 chunks, 516,964 raw / 157,952 gzip bytes versus 516,722 raw / 157,493 gzip before the slice; `culori` is dev-only and absent from runtime bundles. Lighthouse results are local synthetic evidence, not assistive-technology or field-performance proof.
- Protected Web routes now use an explicit `mock-session` / `client-session` resolution seam. Same-origin mock sessions are verified in middleware and bootstrapped by the protected Server Component into an isolated provider-owned Zustand store, removing the hydration-time `/auth/me` request and loading screen; real API and production proxy modes bypass mock-cookie enforcement and retain one deduplicated browser resolution. A production browser replay loaded `/styleguide` with zero `aria-busy` regions and a request trace containing the document plus 12 static chunks but no `/auth/me`; production proxy mode returned the protected shell with HTTP 200 instead of the former mock-cookie redirect. The client entry remained effectively flat at 13 chunks, moving from 516,964 to 517,931 raw bytes (0.19%) and from 157,952 to 158,179 gzip bytes (0.14%). These are local production-build and request-trace measurements, not field Core Web Vitals.
- Client-owned auth resolution now distinguishes session absence, access denial, and availability failure: only `401` / `AUTH.UNAUTHORIZED` redirects to login, `403` / `AUTH.FORBIDDEN` blocks content without a login loop, and network, timeout, rate-limit, `5xx`, malformed, or unknown failures become a localized `unavailable` state with one explicit deduplicated retry path. Successful external JSON and signed mock payloads share a runtime `AuthUser` guard, so malformed `2xx` data cannot create `authenticated` with an absent or unsupported user. Production browser runs proved `503` stayed on `/styleguide`, preserved the session as unknown, issued one initial `/auth/me` plus exactly one request per click, and logged zero browser errors; `403` stayed on the protected URL with one alerting `main`, while `401` alone redirected to `/login?returnUrl=%2Fstyleguide`. The recovery and validation code kept the route entry at 13 chunks, moving from 517,931 to 520,006 raw bytes (0.40%) and from 158,179 to 158,830 gzip bytes (0.41%). These are local production-build and controlled-proxy measurements, not field reliability metrics.
- Auth entry mutations now validate every successful login, registration, current-session, and logout payload at runtime; normalize standard `error_code` evidence before HTTP status fallbacks; associate backend field keys with reviewed local copy; and treat logout `401` as an idempotent completion while preserving state for unknown outcomes. QueryClient instances are provider-owned, locally presented failures declare typed `errorHandling: 'local'` ownership, writes do not retry without endpoint idempotency evidence, and the global fallback no longer displays raw backend messages. A controlled production proxy proved malformed-login `200`, invalid-credential `401`, and registration `422` each issued exactly one request, stayed on the originating route, rendered one local alert, emitted no global notification or raw backend detail, and logged zero browser warnings/errors; registration also exposed three `aria-invalid` controls with stable descriptions. Against baseline commit `f8b5de2`, `/login` and `/register` stayed at 9 initial chunks and moved from 396,582 to 400,724 raw bytes (1.04%) and from 124,368 to 125,596 gzip bytes (0.99%). These are local production-build and controlled-fault measurements, not field reliability metrics.
- Web i18n defaults now flow through typed env config and shared locale constants instead of duplicated hardcoded values.
- Web request locale detection is isolated in `src/i18n/locale-resolution.ts` with unit tests for cookie, `Accept-Language`, and default fallback behavior.
- Web environment access is guarded by `src/test/env-contract.test.ts`: `src/config/env.ts` resolves public values without a schema-library runtime, `src/config/env-validation.ts` keeps Zod validation server-only, `src/config/server-env.ts` owns secrets and mock runtime switches, and production requires `SESSION_SECRET` only when the mock BFF is explicitly enabled. Production browser chunks contain neither server-only names nor Zod.
- Root verification is split into `make governance` for scaffold guardrails and `make check` for governance plus API/Web verification tiers. CI also calls `make governance` for the root governance job. `run-tiers.sh` prints failing command exit codes, full log paths, and configurable log tails for faster repair loops.
- External GitHub Actions are pinned to reviewed full commit SHAs, use Node 24-compatible releases,
  and run with explicit token permissions. The runner and update contract lives in [`CI.md`](CI.md),
  while `.agents/skills/luas-framework-review/scripts/check-ci-actions.py` prevents movable refs,
  unreviewed action repositories, unsafe triggers, runtime drift, and duplicated pnpm version authority.
- The root `luas-framework-review` skill now defines the long-running review loop.
- `luas-framework-review` can now generate optional HTML architecture review reports in `$TMPDIR` for multi-candidate or cross-turn recommendations.
- HTTP contract changes now have a dedicated root `contract-evolution` skill that orders changes through `contracts/`, API behavior, Web services, mock BFF behavior, and verification.
- Vocabulary and boundary decisions now have a dedicated root `domain-modeling` skill that routes new terms to `CONTEXT.md`, ADRs, local docs, skills, or nowhere.
- Luas diff review now has a dedicated root `luas-code-review` skill that separates Standards findings from Spec findings.
- Bugs and contract-sensitive regressions now have a dedicated root `tdd-regression` skill that requires a failing test before production fixes.
- Downstream extraction now has a dedicated root `downstream-app-extraction` skill with a product-leakage scan helper for keeping product behavior out of the source scaffold.
- Scaffold surfaces are cataloged in [`SCAFFOLD_SURFACES.md`](SCAFFOLD_SURFACES.md) with downstream actions and verification by surface type.
- Skill governance now has a dedicated 30/60/90-day and long-term plan in [`SKILL_GOVERNANCE_PLAN.md`](SKILL_GOVERNANCE_PLAN.md).
- High-signal docs and every non-template `SKILL.md` are guarded by `.agents/skills/luas-framework-review/scripts/check-vocabulary.sh` and CI.
- Local Markdown links across docs and agent guidance are guarded by `.agents/skills/luas-framework-review/scripts/check-doc-links.py` and CI.
- API package boundary drift is guarded by `.agents/skills/luas-framework-review/scripts/check-api-boundaries.sh`, with any current exceptions documented in [`../api/docs/PACKAGE_BOUNDARIES.md`](../api/docs/PACKAGE_BOUNDARIES.md).
- API boundary baseline exceptions are currently zero. `internal/domain` is guarded as standard-library-only, starter registry interfaces now live in `internal/starter/assembly` instead of the old top-level starter contract package, `pkg/support` no longer owns the Luas startup banner, app-specific path helpers, debug dump/timing helpers, generic manager/pipeline pattern helpers, generic control-flow/retry/Optional helpers, generic conditional wrappers, broad string/random helpers, broad collection/map helpers, or mutating dot-notation data helpers, and the remaining `pkg/support` exported surface is guarded as `Blank`, `Filled`, `DataGet`, and `DataHas`; `pkg/response` no longer imports `internal/domain`, `internal/capabilities/ai` no longer imports `internal/infra/http`, and `internal/capabilities/workflow` no longer imports `internal/infra/config`, `internal/infra/retry`, `internal/infra/schedule`, or `internal/infra/queue`.
- Branch and release governance now lives in [`BRANCHING_AND_RELEASES.md`](BRANCHING_AND_RELEASES.md): `dev` and `dev-c` are testing branches, deployment branches are CI-managed triggers, and `release/*` or accepted feature PRs are the normal path to `main`.
- Branch/release governance is guarded by `.agents/skills/luas-framework-review/scripts/check-branch-governance.sh` and CI so docs stay aligned with deployment branch mappings.
- Scaffold surface classification is guarded by `.agents/skills/luas-framework-review/scripts/check-surface-catalog.py` and CI so the catalog, glossary, and downstream extraction workflow stay aligned.
- Starter business readiness is now reviewed in [`STARTER_BUSINESS_ROADMAP.md`](STARTER_BUSINESS_ROADMAP.md). Optional `organization` includes the complete ownership/member/invitation/context lifecycle; dependent `permission` adds exact grants and access roles; independent `notification` adds idempotent internal publication, user preferences, in-app state, durable email delivery, and a strict Web center. All three are ready when explicitly enabled in both halves; file/asset, settings, usage, billing, webhook, and AI workspace remain planned.

## Candidate Queue

### Completed P0 — Sensitive Telemetry And Diagnostic Output Boundary

The request logger previously appended the complete raw query to `path`, so access tokens and other
query values reached production logs. The local exception center copied Authorization, Cookie,
`X-API-Key`, and query credentials into its debug model and interpolated all dynamic values into HTML
without escaping, making a panic response both a secret disclosure and debug-mode XSS surface. GORM
logging also allowed interpolated parameter values, while arbitrary audit changes/metadata had no
defense-in-depth redaction.

Request logs now keep only the Gin route template, use a bounded sentinel for unmatched routes, and
never collect concrete paths, query values, or bodies. HTTP traces similarly retain the route shape
and typed error summary without concrete paths or free-form error messages, including database
spans. `pkg/redact` applies one
credential-key vocabulary recursively across configured logger context, exception request data,
recent log context, and audit persistence. The exception renderer escapes every dynamic field, SQL
logs parameterize normal and GORM `Scan` traces, and a root governance check blocks drift. Callers
must still avoid secrets in free-form messages and ambiguous keys; automatic redaction is a final
boundary, not a logging API.

On Go 1.25.12, Apple M3 Max, the five-run representative redaction microbenchmark measured
593.5-626.2 ns/op, 400 B/op, and 9 allocs/op for the six-field request-log context; the nested
sensitive context measured 655.7-675.0 ns/op, 744 B/op, and 11 allocs/op. These are host-sensitive
sanitizer costs, not end-to-end request latency or an SLO.

Verification:

- red/green tests across `pkg/logger`, `pkg/errors`, `internal/infra/exception`, HTTP tracing,
  database diagnostics, and audit service
- `cd api && go test ./pkg/redact -run '^$' -bench '^BenchmarkMap$' -benchmem -count=5`
- `python3 .agents/skills/luas-framework-review/scripts/check-sensitive-telemetry.py`
- `make check`

### Completed P0 — API Key Revocation And One-Time Secret Boundary

The previous repository saved a stale full row when recording use, so a concurrent revoke could be
overwritten with `revoked_at = NULL`. Scopes were stored as ambiguous comma-separated text without
grammar bounds, and the Web settings page displayed a fabricated `sk_demo` value with a nonfunctional
regenerate button. Revocation now updates only revocation columns, usage writes only `last_used_at`
for an active stale row, scopes use JSON for new writes, and route guards consume exact normalized
scopes. The browser feature uses fixed authenticated adapter paths or the explicit mock BFF and
keeps create plaintext out of list and mutation caches.

Verification:

- `cd api && go test ./internal/modules/apikey -count=1`
- `cd web && pnpm vitest run src/test/api-key-contract.test.ts src/test/api-key-route.test.ts src/test/api-key-ui.test.tsx`
- `python3 .agents/skills/luas-framework-review/scripts/check-api-key-boundary.py`
- `make check`

### Completed P0 — HTTP Listen and Transport Configuration

`SERVER_HOST` previously changed only the startup banner while `http.Server` always listened on
`:port`, exposing a supposedly loopback-only process on every interface. The server now constructs
its socket address from the resolved host and port, defaults local execution to `127.0.0.1`, and
makes Docker's wildcard bind explicit. The previously inert `SERVER_MAX_HEADER_BYTES` setting plus
read-header and idle deadlines are now applied to `http.Server`. The default 190-second write budget
outlives the 180-second cooperative middleware timeout, and invalid relationships fail before the
server starts.

Verification:

- `cd api && go test ./internal/bootstrap ./internal/infra/config`
- Real server socket inspection for loopback and wildcard binds, plus an oversized-header request.

### Completed P0 — Production Container Runtime Contract

The previous image copied `.env.example` to `/app/.env`, allowing development CORS, debug server
mode, and example infrastructure values to override production-safe code defaults. It also had no
health check, production request logs were directed to an unwritable file instead of container
stdout, and a local `bootstrap.test` expanded the Docker context to 40.99 MB. The image now embeds no
environment file, emits JSON request logs to stdout without a file handler, runs an executable
non-root liveness probe, and has a shared local/CI smoke verifier. Compose now declares itself as a
loopback-only development stack with local-only credentials.

Measured on Docker Desktop, the first full post-change context transfer was 87.65 kB, a 99.79%
reduction from 40.99 MB. Image size moved from 24,942,104 to 24,944,318 bytes (+2,214 bytes, about
0.009%) while adding the health command and runtime contract. These are local image/build-context
measurements, not a registry transfer or multi-platform budget.

Verification:

- `cd api && make container-check`
- `cd api && make compose-check`
- `cd api && docker compose config --quiet`
- `cd api && go test ./internal/infra/console/commands ./pkg/logger ./internal/infra/config`

### P1 — Security Defaults

Problem: API security middleware now has default guardrails and ownership docs, but future changes still need to keep the catalog, kernel tests, and production knobs in sync.

Recommended slice:

1. Keep [`../api/docs/MIDDLEWARE.md`](../api/docs/MIDDLEWARE.md) as the source of truth when moving middleware between default, starter-owned, opt-in, and deployment-owned categories.
2. Add production configuration checks when new middleware knobs are introduced.
3. Keep default kernel tests in sync when guardrails change.

Verification:

- `cd api && go test ./internal/bootstrap/... ./internal/infra/middleware/... ./internal/infra/ratelimit/...`
- `cd api && golangci-lint run ./...`

### Completed P1 — CI Action Runtime And Supply Chain

All external action references across the four root workflows now use reviewed full commit SHAs with
exact release annotations. Checkout remains on v5.0.1 to preserve compatibility with self-hosted
GitHub Actions Runner v2.327.1, while setup-go, setup-node, pnpm/action-setup, and the existing
golangci-lint action use Node 24-compatible releases. Read-only validation workflows keep explicit
`contents: read`; only deployment-branch synchronization retains `contents: write`; and
`pull_request_target` is forbidden for repository-code verification.

The pnpm version now has one authority in `web/package.json` instead of a duplicate workflow value.
The executable `check-ci-actions.py` guard validates every workflow, reviewed repository/SHA/version
triples, action runtime metadata, token-permission declarations, trigger safety, pnpm ownership, and
the runner contract in [`CI.md`](CI.md). Remote acceptance includes annotation inspection because a
green workflow with a runtime-deprecation warning is not considered clean.

Verification:

- `python3 .agents/skills/luas-framework-review/scripts/check-ci-actions.py`
- `make governance`
- `make check`
- GitHub CI, Container Contract, and Skill Self-Test runs, including annotation inspection.

### Completed P1 — Same-Origin Production Auth Adapter

The Web browser session contract now connects to the Go `user` starter through an explicit
server-only adapter instead of pretending the two HTTP contracts are interchangeable. Browser
routes remain on same-origin `/api/auth/*`; the adapter maps fixed Go paths and DTOs, stores the JWT
in an HttpOnly host cookie, resolves protected sessions on the server, preserves canonical status,
`error_code`, `request_id`, and rate-limit evidence, and never exposes bearer credentials to browser
JavaScript. Login and registration reject cross-origin mutations, upstream requests reject redirects
and arbitrary paths, response messages are locally owned, and only a single ingress-validated IP can
reach Go's trusted-proxy boundary. The same-origin guard uses validated `NEXT_PUBLIC_APP_URL` as its
authority, so a public application origin remains valid when Next.js sees an internal reverse-proxy
URL.

The fictitious Web-only `admin` / `member` role was removed because the scaffold does not yet ship a
permission starter. Auth bootstrap now distinguishes authenticated, unauthenticated, forbidden, and
dependency-unavailable states without converting an API outage into a false logout. Stateless logout
is documented honestly: deleting the browser cookie does not revoke an already issued Go JWT.

Against baseline commit `f12e9ca`, `/login` and `/register` remain at 11 client chunks and move from
404,909 to 404,980 raw bytes (+71 bytes, 0.018%), while gzip moves from 127,145 to 127,142 bytes
(-3 bytes). `/styleguide` remains at 13 chunks and moves from 523,031 to 523,102 raw bytes (+71 bytes,
0.014%), while gzip moves from 159,141 to 159,138 bytes (-3 bytes). The server-only adapter therefore
keeps the browser payload effectively flat. These are local production-build measurements, not field
Core Web Vitals.

A real Docker Postgres + Go API + Next.js run proved registration, current-session resolution, and
the protected page at HTTP `200`; invalid credentials at `401`; cross-origin mutation rejection at
`403` without reaching Go; logout followed by `401`; and API outage as `503` while the protected page
remained on its URL in a retryable unavailable state. The browser response exposed neither JWT nor
invented role fields.

Verification:

- `cd web && pnpm exec vitest run src/test/go-api-auth-adapter.test.ts src/test/auth-adapter-route.test.ts src/test/auth-session-cookie.test.ts src/test/auth-route-backend.test.ts`
- `cd web && pnpm type-check && pnpm lint && pnpm build`
- `make governance` and `make check`
- Real Web-to-Go registration, server-resolved session, invalid login, logout, and API-outage flows.

### Completed P0 — Private Authenticated Response Cache Boundary

Auth Route Handlers previously relied on Next.js route dynamism and cookie access without declaring
an intermediary-cache policy. Real production login and current-session responses therefore had no
`Cache-Control` header despite carrying identity state and `Set-Cookie`. Every auth success,
validation error, same-origin rejection, missing-session response, upstream failure, and unavailable
backend now passes through one route-level finalizer that sets `Cache-Control: private, no-store` and
adds `Vary: Cookie`. Organization Route Handlers use the same server-only primitive; active-context
responses merge `Organization-Id` instead of replacing either dimension. Existing request-ID,
rate-limit, and upstream `Vary` headers survive, comparisons are case-insensitive, and `Vary: *`
remains intact.

Against baseline commit `40b2f8e` with the optional organization feature enabled, all 42 static
files and all 40 JavaScript files are byte-identical: static JavaScript remains 1,575,882 bytes with
content-set SHA-256 `0a27629484c61dddc3bf75c1893b3c4f137c3726d54f343cd3decfca9fb84375`.
The 205 server JavaScript artifacts move from 5,799,376 to 5,801,640 bytes (+2,264, 0.039%). These
are local production-build footprint measurements, not a CDN conformance or latency claim. A real
production process returned the policy on login `200`, current session `200`, malformed login `400`,
and cross-origin logout `403`; organization context returned `Vary: Cookie, Organization-Id`.

Verification:

- `cd web && pnpm exec vitest run src/test/private-response.test.ts src/test/auth-route-backend.test.ts src/test/auth-adapter-route.test.ts src/test/api-error-contract.test.ts src/test/mock-bff-route-contract.test.ts src/test/organization-route.test.ts`
- `cd web && pnpm type-check && pnpm lint && pnpm build`
- `make governance` and `make check`
- Production HTTP replay across auth success/error/origin branches and organization context.

### Completed P1 — Typed Configuration Authority

Environment resolution now implements its documented precedence explicitly: process values,
`LUAS_ENV_FILE`, environment-local, local, environment, base file, then typed defaults. Test and
diagnostic reloads remove stale file-owned values, malformed files fail configuration loading, and a
missing explicitly selected file is no longer ignored. The server gives one validated `*Config`
snapshot to both logging and Wire; capability utilities use typed subset loaders rather than reading
environment variables from runtime packages.

The duplicate dot-notation repository, misleading file watcher, no-op config cache commands,
`fsnotify` dependency, fake ClickHouse sink, and no-op log rotation settings have been removed.
Doctor validates `.env.example` structure without treating every optional key as required and treats
provider model IDs as provider-owned values. `make governance` now enforces the authority boundary
through `check-config-authority.py`. Production aliases (`production`, `prod`, `release`) now share
the same validation/defaults, including debug-off Gin release mode and JSON stdout without local
file logging; all migration commands require `--force`, while production
`serve --migrate` is rejected in favor of a serialized pre-deploy job.

Against commit `8aebae8` on Go 1.25.12 with `-trimpath -ldflags='-s -w'`, the server remains exactly
`34,445,106` bytes while its package dependency count falls from 633 to 631. The CLI dependency count
falls from 638 to 636; the richer Doctor diagnostics move the CLI from `34,776,002` to `34,792,546`
bytes (+16,544 bytes, about 0.048%). `go mod why` confirms `fsnotify` is no longer needed. These are
dependency and binary-footprint measurements, not latency claims. The verified production image
moves from `24,944,318` to `24,951,977` bytes (+7,659 bytes, about 0.031%) while preserving its
non-root, health, environment-file exclusion, and graceful-shutdown contracts.

Verification:

- `cd api && go test ./pkg/env ./pkg/logger ./pkg/errors ./internal/infra/config ./internal/bootstrap ./internal/infra/console/commands ./tests/unit`
- `cd api && go test -race ./tests/unit ./internal/infra/config ./internal/infra/console/commands ./internal/bootstrap ./pkg/env ./pkg/logger ./pkg/errors`
- `cd api && env DB_ENABLED=false JWT_SECRET=... AI_ENABLED=false go run ./cmd/luas doctor`
- `make governance`, `make check`, `cd api && make container-check`, and `cd api && make compose-check`

### Completed P1 — Database-Disabled Runtime Degradation

The explicit `DB_ENABLED=false` mode no longer leaves default starter repositories holding an
unsafe nil GORM pointer. User, API key, and audit repositories now return the shared
`domain.ErrServiceUnavailable` sentinel; the HTTP boundary maps it to `503` +
`COMMON.SERVICE_UNAVAILABLE`. Lookup services preserve dependency failure instead of misclassifying
it as not found or invalid credentials, while audit persistence remains best-effort and cannot append
a panic response after an existing 401/4xx envelope. The full-kernel regression covers unauthenticated
mutations plus authenticated user, API key, and audit routes with the database disabled.

Verification:

- `cd api && go test ./internal/bootstrap -run '^TestHTTPKernelDatabaseDisabledDoesNotPanic$'`
- `cd api && go test ./internal/modules/user ./internal/modules/apikey ./internal/modules/audit`
- Real server with `DB_ENABLED=false`, including 401 and 503 JSON envelopes plus readiness.

### Completed P1 — Queue Lifecycle Concurrency

The queue lifecycle race exposed by `go test -race ./tests/unit` is fixed. The memory driver now owns
an explicit close barrier, bounded FIFO state, delayed-task cancellation, and in-flight operation
tracking. Pending delayed deliveries also use a bounded, context-aware backpressure budget instead
of allowing unbounded timer goroutines. `Close` is concurrent-safe and idempotent; blocked producers,
consumers, and workers exit without a send/close race or a closed-channel panic. The contract and
durable replacement boundary are documented in [`../api/docs/WORKFLOW.md`](../api/docs/WORKFLOW.md).

Measured locally on an Apple M3 Max with Go 1.25.12, a 256-byte push/pop round trip changed from
approximately `56 ns/op`, `0 B/op`, `0 allocs/op` to `80 ns/op`, `0 B/op`, `0 allocs/op`. The roughly
42% synchronization cost is retained because it buys deterministic shutdown while still sustaining
about 12.5 million in-process round trips per second. This is evidence, not a CI performance budget.

Verification:

- `cd api && make test-race-critical`
- `cd api && go test ./...`
- `cd api && make benchmark-workflow`

### P1 — Measured Performance Baseline

Problem: Luas now has one measured API dependency/binary baseline and bounded HTTP metric labels, but it does not yet guard API latency, database query behavior, Web route bundles, or Core Web Vitals with repeatable budgets.

The core HTTP middleware portion now has a repeatable metrics-off/metrics-on benchmark and a
steady-state allocation gate. On an Apple M3 Max with Go 1.25.12, the metrics-disabled median moved
from about `1.77 us`, `1,446 B`, and `41 allocs/request` to `1.25 us`, `1,138 B`, and
`18 allocs/request`; the metrics-enabled median moved from about `2.08 us` and `42 allocs/request`
to `1.47 us` and `18 allocs/request`. CI guards only the stable allocation signal with a
`21 allocs/request` ceiling. Host-sensitive timing remains comparison evidence, not an SLO.

Recommended slice:

1. Keep dependency and stripped binary measurements comparable when changing runtime dependencies.
2. Keep the representative API middleware benchmark and allocation budget aligned when the kernel or request metrics change.
3. Add Postgres-backed measurements for query count, allocation, and p95 latency on starter list/write flows before claiming database improvements.
4. Record Web build route output and route-level client bundle evidence before changing provider placement, i18n routing, charts, or analytics.
5. Promote a measurement into CI only after it is stable across runners and has an explicit regression threshold.

Verification:

- `cd api && go test -run '^$' -bench . -benchmem ./internal/bootstrap/... ./internal/infra/metrics/...`
- `cd api && go build -trimpath -ldflags='-s -w' -o /tmp/luas-server ./cmd/server`
- `cd web && pnpm build`

### P1 — Web Hydration Boundaries

Problem: public auth/query/toast leakage, the auth shell, custom root error handling, optional analytics, and client message breadth are now guarded; the remaining shared hydration cost is the i18n/theme runtime itself.

Recommended slice:

1. Keep Query/Auth providers route-scoped instead of returning them to root.
2. Keep `src/test/public-route-boundary.test.ts` aligned with the public route dependency boundary.
3. Keep the auth visual shell server-rendered and `src/test/i18n-runtime-boundary.test.ts` aligned with its client leaves.
4. Keep `src/test/root-runtime-boundary.test.ts` aligned with route-scoped toast, server-rendered optional analytics, and App Router error conventions.
5. Keep `src/test/i18n-client-messages.test.tsx` aligned with global and route-owned client namespaces.
6. Prefer server-rendered labels for new interactive leaves when that removes a client translation dependency cleanly.
7. Review cookie/header-driven i18n separately because it keeps routes dynamic even after provider scoping.

Verification:

- `cd web && pnpm build`
- Bundle comparison before/after.

### P1 — i18n Runtime Boundary

Problem: locale detection currently reads cookies and request headers, which preserves language switching but keeps otherwise public routes dynamic.

Recommended slice:

1. Keep `src/i18n/locales.ts`, `src/i18n/locale-resolution.ts`, `src/config/env.ts`, and i18n docs in sync when adding locales or changing locale detection.
2. Decide whether downstream apps prefer cookie/header locale detection or locale-prefixed static routes.
3. If static public routes become the goal, introduce an explicit route strategy rather than silently disabling locale switching.

Verification:

- `cd web && pnpm type-check`
- `cd web && pnpm build`

### P1 — Demo and Mock Production Guardrails

Problem: mock BFF and demo credentials are excellent for scaffold usability but must be difficult to ship accidentally as product behavior.

Recommended slice:

1. Keep new mock route handlers behind `guardMockBffRoute()`.
2. Return mock success payloads through `apiSuccessResponse()` and errors through the shared error response helpers.
3. Keep the client/server env split and conditional `SESSION_SECRET` requirement covered by `src/test/env-contract.test.ts` when changing mock auth or deployment behavior.
4. Run `src/test/mock-bff-route-contract.test.ts` when adding or deleting mock route handlers.
5. Keep `web/docs/MOCK_BFF.md` current when mock route handlers, demo credentials, or auth session behavior change.
6. Add production configuration tests when adding new demo-only flows.

Verification:

- `cd web && pnpm vitest run`
- `cd web && pnpm build`

### P1 — Scaffold Error Contract Drift

Problem: scaffold-level HTTP status and `error_code` behavior spans `contracts/README.md`, API response constants, Web fallback mapping, and mock BFF behavior; changing one without the others makes downstream apps branch on stale assumptions.

Recommended slice:

1. Keep `.agents/skills/luas-framework-review/scripts/check-error-contracts.py` aligned with the scaffold-level errors documented in `contracts/README.md`.
2. Add new scaffold-level `error_code` values to the contract first, then API response constants, then Web `ApiErrorCode` and status fallback behavior.
3. Keep domain-specific codes out of the scaffold-level table until they become shared HTTP contract behavior.

Verification:

- `python3 .agents/skills/luas-framework-review/scripts/check-error-contracts.py`
- `cd web && pnpm vitest run src/test/error-code-vocabulary.test.ts src/test/api-error-contract.test.ts`
- `make check`

### P1 — Branch and Release Discipline

Problem: shared testing branches are useful for many teams, but they become unsafe when unfinished work and release-ready work are mixed and then merged wholesale into `main`.

Recommended slice:

1. Keep [`BRANCHING_AND_RELEASES.md`](BRANCHING_AND_RELEASES.md) aligned with `.github/workflows/ci.yml` and `.github/workflows/sync-deploy-branches.yml`.
2. Treat `dev` and `dev-c` as mutable testing branches, not release candidates.
3. Assemble release content from `main` using `release/*`, accepted feature PRs, or explicit cherry-picks.
4. Keep deployment trigger branches mechanical and CI-owned.

Verification:

- `make check`
- `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`
- `bash .agents/skills/luas-framework-review/scripts/check-branch-governance.sh`
- Inspect `.github/workflows/sync-deploy-branches.yml` when changing branch names or environment mappings.

### Completed P1 — Additive Optional Starter Assembly

Optional starters previously required manual edits across Wire, routes, migration commands, and
seeder commands, so a downstream app could easily activate only half a business capability. The
typed `OPTIONAL_STARTERS` catalog now keeps `user`, `apikey`, and `audit` as immutable defaults while
adding selected manifests, routes, runtime hooks, migrations, and seeders from one snapshot. Unknown,
duplicate, default, and non-canonical names fail startup. Offline asset resolution now safely omits
typed-nil runtime modules, and the application migrator is populated by the same registry used by
HTTP.

The first optional entry is an `organization` ownership kernel: atomic organization/owner creation,
membership-scoped list/get, owner/admin rename, stable organization errors, audit changes, and an
membership-aware account-deletion guard that prevents orphaned tenants. Its invitation lifecycle adds
manager-scoped create/list/revoke, transactional acceptance, immutable token hashes, explicit email
attempt semantics, stable errors, and audit changes. `organization` means the tenant boundary; it is
not interchangeable with a future workspace concept. Its member lifecycle adds a PII-minimized
directory, owner-only role changes, manager removal and self-leave policy,
transactional ownership transfer, membership audit changes, and account-deletion guards that stop
soft-deleted users from leaving stale membership rows.

At the initial ownership-kernel delivery against baseline commit `98865a7`, default assembly stayed
at 14 routes and seven migrations while the optional starter exposed four additional routes and one
migration. The Darwin/arm64 stripped server moved from 34,445,106 to 34,544,338 bytes (+99,232 bytes,
0.29%); the
module graph remains at 276 and `go.mod` / `go.sum` are unchanged. Five-run HTTP middleware medians
remain effectively flat at 1,207 versus 1,186 ns/request with metrics disabled and 1,432 versus
1,410 ns/request with metrics enabled; all runs retain 18 allocations/request. These host-local
timings are regression evidence, not an SLO. A real PostgreSQL run exercised the full ownership
flow and an 8 -> 7 -> 8 migration rollback/reapply cycle.

The invitation slice keeps the default at 14 routes and seven migrations and raises
`OPTIONAL_STARTERS=organization` to eight additional routes and two migrations. It adds no Go module
dependency; token storage, active-invitation uniqueness, and membership creation are database-owned
transactional invariants rather than process-local checks.

Against the immediate pre-slice commit `33de03e`, the stripped `CGO_ENABLED=0` Darwin/arm64 server
moves from 35,265,602 to 35,350,818 bytes (+85,216 bytes, 0.24%). The module graph remains at 276,
and `go.mod` / `go.sum` remain unchanged. Five-run HTTP middleware medians move from 1,246 to 1,238
ns/request with metrics disabled and from 1,461 to 1,465 ns/request with metrics enabled; both retain
18 allocations/request. These are host-local regression baselines, not an SLO; the invitation path
is intentionally off the default request hot path.

Container verification now preserves artifact identity: standalone `make compose-check` rebuilds the
current worktree, while CI may explicitly reuse only the image verified by the preceding container
step. An `OPTIONAL_STARTERS=organization` PostgreSQL run exercises organization/invitation statuses
`201/201/409/200/204/201` for create, invite, duplicate, list, revoke, and replacement, respectively.
This closes the prior false-green path where a stale local `luas-api:compose-check` tag could bypass
the current source tree.

The member-lifecycle slice keeps the default assembly at 14 routes and seven migrations, raises
`OPTIONAL_STARTERS=organization` from eight to twelve additional routes, and keeps its migration
count at two. No schema change is needed: role and membership transitions fit the existing table,
and member pagination follows `user_id`, which is already the second column of the unique
`(organization_id, user_id)` index. PostgreSQL `FOR UPDATE` locks serialize mutations by locking the
actor membership before the target. A real concurrent transfer produced exactly `200/403`, retained
one owner, and demoted the previous owner to admin; the same run covered directory privacy, role
policy, account-deletion guards, manager removal, self-leave, and owner-leave rejection.
Account deletion now locks the undeleted user row and runs starter guards plus soft deletion in one shared
transaction. Organization creation and invitation acceptance take the same user lock before writing
memberships. A PostgreSQL race between account deletion and organization creation allows only
`201/409` or `204/404`, followed by a direct zero-orphan membership assertion.

Against pre-slice commit `f4213e6`, the stripped `CGO_ENABLED=0` Darwin/arm64 server moves from
35,350,818 to 35,402,146 bytes (+51,328 bytes, 0.145%), and the production image moves from
25,047,624 to 25,088,293 bytes (+40,669 bytes, 0.162%). The module graph remains at 276 with no
`go.mod` or `go.sum` change. Five 2-second HTTP middleware runs move median time from 1,238 to 1,247
ns/request with metrics disabled (+0.7%) and from 1,428 to 1,460 ns/request with metrics enabled
(+2.2%); all runs retain 18 allocations/request. These host-local results are regression evidence,
not an SLO or field-performance claim, and the new organization routes remain outside the default
runtime surface.

The active-context slice adds the named `organization_context` middleware and
`GET /v1/organization-context`. `Organization-Id` is a request-scoped selection, not an authority:
the resolver joins the current membership and organization in one indexed query, binds a typed
value to the request, emits cache-safe `Vary` metadata, and records organization identity for
mutating-route audits. Missing, malformed, duplicate, or overflowing selections fail before
persistence; absent and non-member IDs share `404 ORGANIZATION.NOT_FOUND`. Default CORS now permits
the explicit header, and the Compose gate covers preflight, required selection, owner resolution,
non-member non-disclosure, and current member role against PostgreSQL. No schema migration or Go
module dependency was added. Named middleware lookup now fails route assembly on an unknown name,
preventing an auth or organization-context typo from silently dropping a protection layer.

Against pre-slice commit `c8ef26d`, default assembly remains at 14 routes and seven migrations;
`OPTIONAL_STARTERS=organization` moves from twelve to thirteen additional routes and remains at two
migrations. The stripped `CGO_ENABLED=0` Darwin/arm64 server moves from 35,402,146 to 35,419,282
bytes (+17,136 bytes, 0.048%), and the production image moves from 25,088,293 to 25,097,694 bytes
(+9,401 bytes, 0.037%). The module graph stays at 276 with no `go.mod` or `go.sum` change. Five
2-second core HTTP middleware runs move median time from 1,209 to 1,240 ns/request with metrics
disabled (+2.6%) and from 1,444 to 1,450 ns/request with metrics enabled (+0.4%); all runs retain 18
allocations/request. These are host-local regression measurements, not an SLO or field-performance
claim. The active-context database test separately proves one query for both member success and
non-member denial.

The first browser ownership slice adds optional organization list/create/select/context/rename
surfaces without making tenant selection ambient state. The selected organization stays in the URL,
and every context-sensitive same-origin request carries an explicit `Organization-Id`. Development
uses a bounded in-memory mock; production uses fixed route adapters with an HttpOnly API token,
same-origin mutation checks, private no-store responses, safe header forwarding, bounded request and
upstream response bodies, request IDs, and canonical API envelopes. Authentication precedes resource,
context-header, and payload validation, while cross-origin writes are rejected before session access.
The generic `API_*` adapter
configuration replaces auth-only naming while retaining one deprecation window for legacy aliases.
Runtime schemas reject unsafe numeric IDs, permissive timestamps, malformed pagination, and names
outside the API's 2-100 Unicode-code-point contract. The optional Web feature remains disabled by
default, and no package or lockfile dependency changed.

Against baseline commit `122f535`, the three new organization routes raise the static JavaScript
inventory from 1,437,358 to 1,575,882 bytes (+138,524, 9.64%). Direct route imports, named
`zod/mini` imports, and narrower contract schemas reduced total static JavaScript by 228,919 bytes
(12.68%) from the first implementation. The existing console route union moves only from 988,060
raw / 303,652 gzip bytes to 992,030 raw / 305,659 gzip bytes (+3,970 / +2,007); the optional
organization directory adds 67,077 raw / 20,305 gzip bytes when visited. The lazily loaded switcher
inventory is 51,287 raw / 15,762 gzip bytes. Gzip values use Node zlib level 9. These are local
Next.js production-manifest and chunk measurements, not network transfer or field Core Web Vitals.
Production-browser runs completed the
create, URL selection, context verification, rename, and reload flow at 1280 px and 390 px with no
console warnings/errors and no page-width overflow.

The completed browser lifecycle adds member directory/role/remove/leave controls, atomic ownership
transfer, invitation create/list/revoke, and manually entered token acceptance through explicit
same-origin allowlist handlers. Member and invitation success schemas are strict, so an accidental
member email or invitation token fails as `CLIENT.INVALID_RESPONSE` before cache or render. The
mock store now carries an authenticated actor and the production role matrix instead of a global
implicit owner; its injectable token seam exists only in store tests and no route response or log
reveals the secret. Forty-six focused contract, route, static-boundary, and UI tests cover the Web
lifecycle. A local standalone production run returned `200` for login, the organization page, and
the PII-minimized three-member directory, `201` for token-free invitation creation, and `204` for
revoke. Desktop and 390 px browser runs exercised the profile/member/invitation tabs and invitation
dialog with no page-level horizontal overflow; the only console warning was the expected missing
development `SESSION_SECRET` warning, while the production replay used an explicit secret.

Against immediate baseline commit `784d008`, an organization-enabled Next.js 16.2.9 production
build moves the complete static JavaScript inventory from 40 chunks / 1,575,734 raw / 487,993 gzip
bytes to 42 chunks / 1,635,047 raw / 503,624 gzip bytes (+59,313 / +15,631, or 3.76% / 3.20%). The
organization-detail client-reference set moves from 16 chunks / 494,643 raw / 155,304 gzip bytes to
17 chunks / 550,102 raw / 169,996 gzip bytes (+55,459 / +14,692, or 11.21% / 9.46%). Gzip values
use Node zlib level 9. No package or lockfile dependency changed. These are local build-manifest
measurements, not network transfer or field Core Web Vitals.

The independent notification slice adds a user-scoped optional starter rather than a global process
manager. Internal publication is transactional and idempotent per `(user_id, idempotency_key)`;
preferences default on but cannot suppress required channels; in-app state uses a monotonic
high-water update; and durable email deliveries use bounded database leases, stable provider
idempotency keys, compare-and-set completion, five-attempt terminal failure, destination hashes, and
privacy-minimized storage and logs. The API exposes six authenticated routes and three migrations,
while the default route and migration sets remain unchanged. A signal-aware
`notification:work` command owns the replaceable delivery loop.

The Web feature adds strict same-origin adapters and a user-isolated mock BFF store, validates every
response, rejects absolute, protocol-relative, encoded-separator, backslash, control-character, and
malformed action URLs, and renders provider text without HTML interpretation. The center polls only
unread status while closed and loads the list and preferences on demand. A desktop and 390 x 844
browser run exercised two-user-safe mock login, two notification records, unread-to-read transition,
preference replace and reload, keyboard access, and mobile layout with no page-width overflow. The
only console warning was the expected development fallback for an unset `SESSION_SECRET`.

With Next.js 16.2.9, disabling and enabling notification both retain 13 base `/console` entry chunks:
419,816 raw / 132,425 gzip bytes disabled and 419,783 raw / 132,420 gzip bytes enabled. Selecting
notification adds a separate two-chunk async boundary of 78,195 raw / 22,891 gzip bytes; an
unselected downstream app does not receive that feature inventory. Gzip values use Node zlib level
9. These are local production-build measurements, not network transfer or field Core Web Vitals. A
real PostgreSQL Compose run produced notification statuses
`200/200/422/404/200/200/200`, a worker result of
`failed:1:EMAIL.NOT_CONFIGURED:64`, and a `3 -> 0 -> 3` migration rollback/reapply cycle.

Verification:

- `cd api && go test ./...`
- Targeted race-enabled user, organization, bootstrap, and feature tests
- Targeted Web adapter, body-boundary, optional-feature, organization-contract, route, and UI tests
- Web production builds with the optional organization feature enabled and disabled
- Browser organization create/select/context/rename/member/invitation flows at desktop and mobile widths
- Browser notification list/read/preferences flows at desktop and mobile widths
- Standalone production login, member-directory, invitation-create, and invitation-revoke HTTP flow
- Disabled/enabled `go run ./cmd/luas route:list` comparison
- Real PostgreSQL invitation/member HTTP flows plus ownership-transfer and account-deletion races
- Real PostgreSQL notification HTTP, worker-failure, privacy-schema, and migration rollback/reapply flow
- `make governance` and `make check`

### P1 — Starter Business Readiness

Problem: the current default starter set is useful for auth, API keys, and audit, but most new SaaS, internal-tool, and developer-product projects also need reusable multi-user ownership, authorization, invitations, notification preferences, files, settings, usage, and integration flows.

Recommended slice:

1. Use [`STARTER_BUSINESS_ROADMAP.md`](STARTER_BUSINESS_ROADMAP.md) as the starter readiness matrix before adding new route-owning behavior.
2. Keep the complete optional `organization` lifecycle and its production/mock adapter parity under executable contract and browser regression coverage.
3. Keep the delivered `permission` starter optional, organization-dependent, exact-match, fail-closed, and covered by `.agents/skills/luas-framework-review/scripts/check-permission-boundary.py`.
4. Keep the delivered `notification` starter user-scoped, idempotent, lease-driven, privacy-minimized, and covered by `.agents/skills/luas-framework-review/scripts/check-notification-boundary.py`.
5. Build file/asset next; promote any starter into the default scaffold only after its deletion path, contract, security defaults, and downstream value are proven.

Verification:

- `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`
- `PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-doc-links.py`
- `make governance`

### P2 — Architecture Vocabulary Cleanup

Problem: some docs still use framework/module/feature/console language loosely.

Recommended slice:

1. Search docs and comments for flagged ambiguities from `CONTEXT.md`.
2. Update only user-facing or agent-facing text first.
3. Avoid broad code renames unless they remove real confusion.

Verification:

- `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`
- `git diff --check`
- Targeted `rg` scans for old terminology.

### P2 — Package and Seam Deepening

Problem: packages such as `support`, `utils`, `pkg/errors`, `pkg/response`, and `internal/starter/assembly` should be reviewed for shallow-module drift. The API boundary check also records current reverse-import exceptions so they can be migrated deliberately.

Recommended slice:

1. Pick one seam from [`../api/docs/PACKAGE_BOUNDARIES.md`](../api/docs/PACKAGE_BOUNDARIES.md) or the package list above.
2. Apply the deletion test.
3. Either document why the seam is valid, deepen/rename it, or remove one baseline exception.
4. For `internal/capabilities/workflow`, guard the now-clean boundary by keeping queue, retry, and scheduler primitives workflow-owned; only `internal/infra/*` compatibility packages may wrap them.
5. Keep `pkg/support` small by requiring new exported helpers to land at the starter, capability, or runtime seam that owns the behavior.

Verification:

- `bash .agents/skills/luas-framework-review/scripts/check-api-boundaries.sh`
- Targeted tests for the selected seam.
- Full affected-half type/lint/test command.

### P2 — AI Workflow Router

Problem: root, API, and Web skills are useful but can be hard to choose during broad framework work.

Recommended slice:

1. Use `luas-framework-review` as the router for global optimization.
2. Add pairings from relevant half-specific skills back to this root skill only where useful.
3. Keep descriptions short to avoid context bloat.

Verification:

- `.agents/skills/scripts/validate-skill.sh --all`
- Skill count checks in `.agents/skills/README.md`.

### P2 — Skill Governance Plan

Problem: skill semantic drift is now guarded across every non-template `SKILL.md`, but the skills still need steady cleanup as Luas vocabulary and architecture rules evolve.

Recommended slice:

1. Follow the 30-day plan in [`SKILL_GOVERNANCE_PLAN.md`](SKILL_GOVERNANCE_PLAN.md).
2. Continue cleaning Web skill terminology around `mock BFF`, `(protected)`, `console`, and feature structure.
3. Continue cleaning API skill terminology around response/domain error mapping.
4. Keep vocabulary checks aligned with newly flagged ambiguity patterns from `CONTEXT.md`.

Verification:

- `bash .agents/skills/scripts/validate-skill.sh --all`
- `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`
- `git diff --check`

### P2 — Root Governance Entry Point

Problem: Luas now has multiple root guardrails for vocabulary, docs, contracts, surfaces, branch discipline, package direction, and skill metadata; if they stay as separate remembered commands, agents and humans will eventually run only part of the governance set.

Recommended slice:

1. Keep `make governance` as the single local entry point for root guardrails.
2. Keep CI's root governance job calling `make governance` instead of duplicating the command list.
3. Keep `make check` running `make governance` before API and Web verification tiers.
4. Add new root guard scripts to `make governance` when they become stable enough for CI/local use.
5. Keep task-specific checks, such as downstream product leakage patterns, outside the default target unless they can run without product-specific input.

Verification:

- `make governance`
- `make check`

### P2 — Documentation Link Integrity

Problem: Luas relies on AGENTS files, skills, architecture docs, and feature/starter guides as navigation rails; stale local links make both humans and agents choose the wrong seam.

Recommended slice:

1. Keep `.agents/skills/luas-framework-review/scripts/check-doc-links.py` aligned with the docs and generated/vendor exclusions.
2. Run it whenever Markdown docs, skill bodies, AGENTS files, or README navigation change.
3. Prefer fixing stale links over widening exclusions, except for intentional templates and generated/vendor trees.

Verification:

- `python3 .agents/skills/luas-framework-review/scripts/check-doc-links.py`
- `bash .agents/skills/scripts/validate-skill.sh --all`
- `git diff --check`

### P2 — Downstream Extraction Guardrails

Problem: Luas now has a downstream extraction workflow, but future scaffold changes still need to keep examples, devtools, mock BFF routes, console surfaces, and product-specific behavior clearly classified.

Recommended slice:

1. Keep [`SCAFFOLD_SURFACES.md`](SCAFFOLD_SURFACES.md), `downstream-app-extraction`, `CONTEXT.md`, `web/docs/MOCK_BFF.md`, `api/docs/ADDING_MODULE.md`, and `web/docs/ADDING_FEATURE.md` aligned.
2. Run the product-leakage helper with task-specific identifiers before committing scaffold-mode changes that touched downstream examples or docs.
3. Keep surface classification checks in CI; avoid baking product names into Luas.

Verification:

- `python3 .agents/skills/luas-framework-review/scripts/check-surface-catalog.py`
- `bash .agents/skills/scripts/validate-skill.sh --all`
- `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`
- `bash .agents/skills/downstream-app-extraction/scripts/check-downstream-contamination.sh --expected-origin git@github.com:zgiai/luas.git --pattern "<task-product-identifier>"`

### P2 — Architecture Review Reports

Problem: architecture report generation now exists, but it should stay optional and lightweight so broad reviews do not become process-heavy.

Recommended slice:

1. Use the report helper when a review compares multiple architecture candidates or needs cross-turn continuity.
2. Keep generated reports in `$TMPDIR` unless the user explicitly wants a committed artifact.
3. Iterate the report fields only when real review runs reveal missing evidence.

Verification:

- generate a sample report in `$TMPDIR`
- `bash .agents/skills/scripts/validate-skill.sh --all`
- `bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh`

## Iteration Rules

- Pick one candidate per turn unless the changes are purely documentary and tightly related.
- Update `CONTEXT.md` when new vocabulary is introduced.
- Update `contracts/README.md` before changing API/Web contract behavior.
- Prefer tests at public seams over implementation-coupled tests.
- Record verification commands in the final report.
- Leave deferred candidates visible so the long-running task keeps continuity.
