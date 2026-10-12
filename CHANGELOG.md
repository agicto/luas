# Changelog

Notable changes to FastBuild. Downstream migration steps for each change are in
[`UPGRADING.md`](UPGRADING.md); the long-running quality record is in
[`docs/FRAMEWORK_QUALITY_ROADMAP.md`](docs/FRAMEWORK_QUALITY_ROADMAP.md).

## Unreleased

## v0.25.0 — 2026-10-12

### Added

- **Plugin registry.** Reviewed capabilities distributed as source: `plugin:search` and
  `plugin:info` (with `--format=json` for agents), `plugin:add` (checksum-verified, atomic with
  rollback, `--dry-run`, imports rewritten to the project's module), `plugin:status`, and
  `plugin:index` for maintainers. Installs are recorded in `fastbuild.plugins.lock`. Projects created
  with `make init-project` install from the FastBuild release they started from.
- **`knowledge-base` plugin**, the first official plugin: organization knowledge articles with
  full-text search and substring search that works for Chinese.
- **`plugin-integration` skill** that teaches coding agents to find, install, integrate, and verify
  plugins.
- Every official plugin is installed into a copy of the API, built, and tested in tier 1, and its
  contract fragment is merged and linted by `make contract-check`.
- [`docs/VISION.md`](docs/VISION.md), [`docs/ROADMAP.md`](docs/ROADMAP.md), and the
  [plugin registry plan](docs/plans/plugin-registry.md).
- **Slots and the plugin lifecycle.** Core files mark named extension points
  (`fastbuild:slot <name>`, listed by `plugin:slots`); plugins and `make:module` edit core files only
  there. The lock file records every file checksum, slot line, and contract addition, so
  `plugin:update` replaces untouched files and merges edited ones three ways, `plugin:remove` restores
  core files and `openapi.yaml` byte for byte, and `plugin:doctor` reports unfinished integration.
  `plugin:pack` copies edits back into the registry for plugin authors.
- **Plugin kinds and templates.** Plugins are `feature`, `theme` (one at a time), `block`, or
  `template`; a template installs a set of plugins together or not at all, and
  `make init-project TEMPLATE=<name>` starts a project from one.
- **Official plugins:** `tags` and `comments` (organization-scoped, with code-owned resource-type
  seams, If-Match concurrency, and audit without content; comments notify @mentions through the
  notification starter); themes `theme-enterprise`, `theme-warm`, and `theme-midnight` for Web and
  Admin; blocks `block-dashboard-stats`, `block-data-table`, `block-settings-page`,
  `block-pricing-table`, `block-empty-state`, `block-auth-card`, `block-onboarding-steps`, and
  `block-landing-hero`; templates `knowledge-portal`, `saas-app`, and `internal-tool`.
- **`project:upgrade --to=<release>`** merges a newer FastBuild release into an initialized project:
  untouched files are replaced, edited ones merged three ways, local deletions kept, and both releases
  adapted to the project's module path and identity first.
- **`fastbuild mcp`** serves read-only project and plugin tools to coding agents over the Model
  Context Protocol. `plugin:search` understands sentences in any language and filters by `--kind`;
  `registry/evals/search-cases.tsv` holds the plugin-selection evaluation.
- **Community registries.** Any GitHub repository with a `registry/` can be a source; sources other
  than FastBuild releases and local directories need `--trust`.
- **Shared rate limiting.** `RATE_LIMIT_STORE=postgres` shares the global and authentication budgets
  across replicas through an UNLOGGED table with one atomic upsert per decision and hashed keys;
  `RATE_LIMIT_STORE_TIMEOUT` and `RATE_LIMIT_STORE_FAILURE=local|open|closed` control outages, and
  `ratelimit:prune` removes expired buckets (ADR 0015).
- **Plugins workflow** and `scripts/verify-plugin.sh`: every official plugin is installed into a copy
  of the checkout, integrated, built and tested on each half it touches, checked by `plugin:doctor`,
  and removed, on registry changes and nightly. Web registry items are type-checked, linted, and
  tested by `pnpm lint`; theme presets pass the Web and Admin contrast guards.
- `FASTBUILD_IGNORE_ENV_FILES=true` skips every env file layer for tools.
- The 1.0 compatibility promise in [`docs/VERSIONING.md`](docs/VERSIONING.md).

### Changed

- `merge-fragment` accepts `x-error-codes`, appended to the `ErrorCode` enum; the error-contract
  check counts error codes declared in a module's `errors.go`, so plugins never edit core files.
- `make init-project` removes `registry/` from new projects.
- Admin Console design tokens moved to `admin/src/styles/tokens.css`; the default muted and
  destructive text colors are slightly darker to meet WCAG AA contrast.
- Commands with `--format=json` no longer print a completion line after the JSON.
- CI runs only the checks a pull request can break; Node 24, PostgreSQL 15/17/18, containers,
  end-to-end, and the plugin matrix run after merge and nightly (see `docs/CI.md`). The unused
  `dev`/`dev-c` deployment sync workflow is removed, and no workflow writes to branches.
- The tracked-artifact check keeps the scaffold free of installed plugins and rejects unmerged
  `*.fastbuild-new` files and unresolved update conflict markers.

### Deprecated

- `plugin:list`, which lists external CLI commands, is renamed `command:list`; the old name remains
  an alias until a later minor release.

## v0.24.0 — 2026-10-11

### Changed

- **Luas is now FastBuild**, maintained at [`fastagi/fastbuild`](https://github.com/fastagi/fastbuild).
  The Go module is `github.com/fastagi/fastbuild/api`, the CLI is `fastbuild`, environment
  variables use the `FASTBUILD_` prefix, and API keys start with `fastbuild_`. See `UPGRADING.md`
  for the full mapping.

### Removed

- Offset pages (`page`) on `GET /v1/audit-logs` and `GET /v1/operator/audit-logs`, deprecated in
  v0.23.0. Both endpoints return keyset pages only; a `page` parameter returns
  `400 COMMON.INVALID_INPUT`. The Go `ListForUser`, `ListAuditLogs`, `FindByUserID`, and `FindAll`
  audit methods are gone with them.

## v0.23.0 — 2026-10-10

### Added

- `DB_QUERY_EXEC_MODE` opts into pgx statement caching (`cache_statement`) for deployments that
  migrate in a maintenance window; the default stays `simple_protocol`. See `api/docs/DATABASE.md`.
- `SERVER_DIAGNOSTICS_ADDR` serves Go runtime profiles on a separate loopback-only listener; off by
  default.
- Cursor pages for `GET /v1/audit-logs` and `GET /v1/operator/audit-logs`: send `cursor` (empty for
  the newest page) and follow `meta.next_cursor`. They skip the per-page `COUNT(*)`; on a million
  audit rows the first page served about 38 times more requests and deep pages fell from up to
  0.5 s to about 2 ms. The Admin Console audit page uses them.
- `AUDIT_WRITE_MODE` (`async` by default, or `sync`) and a `ShutdownModule` assembly seam that the
  HTTP kernel calls after draining requests and before closing the database.
- The nightly k6 baseline seeds one million audit rows and budgets keyset history reads.

### Changed

- Request audit records are written in batches after the response by default, raising write-request
  throughput 50–70%. A record can take about 50 ms to appear in history; graceful shutdown writes
  every queued record.
- The `(user_id, id)` audit index replaces the single-column `user_id` index.
- TypeScript 6 in Web and Admin; lucide-react 1.x; React 19.3 and the Admin TanStack, i18next, and
  Zod minor updates; `x/crypto`, `x/sync`, and `gin-contrib/cors` updates unlocked by Go 1.26.
  golangci-lint 2.14.0 in CI.
- `DB_MAX_IDLE_CONNS` now defaults to `DB_MAX_OPEN_CONNS` instead of 10. The small idle pool
  reopened PostgreSQL connections whenever concurrency dipped; under load, p99 latency on the
  starter read paths fell from about 100 ms to about 10–20 ms and throughput rose about 40%.

### Security

- Go 1.26.9 replaces Go 1.25.13, which no longer receives fixes, closing 13 standard-library
  advisories (html/template, net/http, and others); `golang.org/x/net` 0.60.0 closes four more.
- `seroval` 1.6.3 (critical and high advisories, reached through TanStack Query devtools) and
  `source-map-js` 1.2.2 are pinned through pnpm overrides.

### Deprecated

- Offset pages (`page`) on `GET /v1/audit-logs` and `GET /v1/operator/audit-logs`; use `cursor`.

### Fixed

- Profile, profile update, and password change answered any database failure with
  `404 USER.NOT_FOUND`; only a missing account does now, and other failures are server errors.

## v0.22.0 — 2026-10-04

### Added

- `make init-project` (`fastbuild project:init`) renames a fresh scaffold copy: Go module path, package
  and Compose project names, display name, repository URL, and the default starter selection for the
  API, Web, Admin, and `make dev`. It records the source version in `.fastbuild-project.json`.

- Tag-driven releases: pushing `vX.Y.Z` runs `release.yml`, which requires the tag on `main` with
  a changelog section, builds and scans both images, attests the dependency and image SBOMs, and
  publishes the GitHub release. `make release-check VERSION=vX.Y.Z` verifies `main` before tagging.
- [`docs/VERSIONING.md`](docs/VERSIONING.md) defines version numbers and the deprecation process.
- [`deploy/kubernetes`](deploy/kubernetes/README.md): a hardened kustomize reference baseline for
  the API, workflow worker, Web, the pre-deploy migration job, and per-starter workers and
  retention CronJobs, validated against Kubernetes 1.30.
- `make perf` and a nightly `perf.yml` run a k6 baseline against a release build and PostgreSQL,
  failing when login, readiness, profile, API key, or audit log reads leave their p95 budgets; see
  [`api/docs/PERFORMANCE.md`](api/docs/PERFORMANCE.md).
- `make:module` writes an OpenAPI fragment for the generated routes; `corepack pnpm
  merge-fragment` adds it to `contracts/openapi.yaml` without reformatting the file, so a new
  starter passes the two-way route check without hand-written contract entries.
- The starter table in `api/internal/modules/README.md` and the skill index are generated from the
  starter manifests and skill metadata (`make starter-catalog`,
  `render-skill-index.py --write`), and checks fail when either drifts.
- A Web Playwright smoke test signs in through the API adapter; `e2e.yml` runs it next to the
  Admin Console suite.
- An `api/.env.example` drift test fails when configuration reads a variable the example omits.

### Changed

- The starter-catalog and migration-review checks read the Go module path from `api/go.mod`, so
  they keep working after a rename.
- With `LOG_JSON=true`, every API log line is JSON: GORM query errors and slow queries are
  structured (`database.query_failed`, `database.slow_query`), `log/slog` records go through the
  platform logger, the start-up banner is suppressed, and server and worker start lines are
  structured events.

### Security

- The login subject budget counts failed sign-ins per account: a correct password clears it and
  the username and email of one account share it.
- Sessions record their sign-in audience; operator routes reject sessions from the public login.
- Password-reset lookup and delivery run after the response, so its timing no longer reveals
  whether an account exists.
- A production operator cookie name must keep the `__Host-` prefix; plugin discovery times out.

### Fixed

- The workflow worker's PostgreSQL queue metrics no longer stop reporting while the queue is
  empty; a NULL oldest-task timestamp failed the scan.

### Removed

- The deprecated `ROLE.NOT_FOUND` error code, announced in v0.21.0; access roles use
  `PERMISSION.ROLE_NOT_FOUND`.

## v0.21.1 — 2026-10-04

### Fixed

- The Web BFF answered an invalid notification filter with `422 COMMON.VALIDATION_FAILED`; it now
  returns `400 COMMON.INVALID_INPUT`, as the API does.
- The API Compose verification rolled back only the newest migration and so checked the wrong
  table once the workflow-task migration was added; it now rolls back through the starter's own
  migration. The Container workflow passes again.

## v0.21.0 — 2026-10-04

The first release since v0.20.0. It turns the Admin Console into an operator console, makes every
HTTP operation machine-checkable, and adds a one-command local stack.

### Added

- **One-command development.** `make dev` starts PostgreSQL in Docker and runs the API, the Web
  application, and the Admin Console on the host with every optional starter, migrations, seed data,
  and a seeded platform operator. A dev container covers Codespaces and local containers.
- **Platform operator console.** The optional `operator` starter issues an HttpOnly Admin session
  with Origin and CSRF enforcement. Operators manage users, read the global audit log, edit app
  settings, and see system status, and, with the matching starters, inspect organizations and
  members, diagnose and replay webhook deliveries, and review the notification delivery ledger.
- **Complete OpenAPI contract.** All 92 operations are described; the route check fails in both
  directions, and Web and Admin assert at compile time that their schemas accept every documented
  body.
- **End-to-end tests.** Playwright drives the Admin Console against the real API; CI starts the
  stack with `make dev`.
- **Durable task operations** (`workflow:tasks`, `retry`, `cancel`, `prune`), PostgreSQL durable
  tasks, `make:module` generating a wired starter, starter-owned error mappings, a shared
  `FakeMailer`, starter selection commands, bounded audit retention, and a static TanStack Admin
  shell.

### Changed

- Body validation failures return `422 COMMON.VALIDATION_FAILED` with field errors; invalid input
  and zero IDs return `400`, as the global contract always stated.
- Released migrations are frozen SQL checked against a golden schema; `make wire-check` guards the
  generated Wire graph.
- The Web setting, usage, and webhook pages tolerate a larger server catalog instead of failing.
- PostgreSQL is the only database; CI verifies PostgreSQL 15 through 18.
- Dependabot runs monthly with grouped security updates; CI cancels superseded runs.

### Security

- Next.js 16.3.8 (three remote-code-execution advisories), Go 1.25.13, grpc 1.83.2,
  `golang.org/x/crypto` 0.55.0, patched transitive npm packages, and a patched Web base image;
  `make dependency-scan` and the image scans report no open findings.
- User profile and sign-in writes are column-scoped, so a racing write can no longer restore an old
  password, re-enable a disabled account, or undelete one.
- Email sign-in resolves to the email owner, and new usernames cannot contain `@`.
- Operator routes reject foreign origins on every method; webhook targets in IPv6 transition ranges
  are rejected; CLI plugins load only from `PATH`.

### Removed

- Unused API packages (`pkg/support`, `pkg/utils`, legacy infra wrappers), the MySQL grammar, the
  domain event system and service locator, and obsolete examples.
