# Changelog

Notable changes to Luas. Downstream migration steps for each change are in
[`UPGRADING.md`](UPGRADING.md); the long-running quality record is in
[`docs/FRAMEWORK_QUALITY_ROADMAP.md`](docs/FRAMEWORK_QUALITY_ROADMAP.md).

## Unreleased

### Added

- `make init-project` (`luas project:init`) renames a fresh scaffold copy: Go module path, package
  and Compose project names, display name, repository URL, and the default starter selection for the
  API, Web, Admin, and `make dev`. It records the source version in `.luas-project.json`.

### Changed

- The starter-catalog and migration-review checks read the Go module path from `api/go.mod`, so
  they keep working after a rename.

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
