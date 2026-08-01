# Contributing To Luas

Luas is an open-source application scaffold. Contributions are welcome when they improve the reusable foundation without turning it into a product-specific application.

## Before You Start

1. Read [AGENTS.md](AGENTS.md) and the `AGENTS.md` inside the area you will change.
2. Read [CONTEXT.md](CONTEXT.md) when global vocabulary or ownership is active.
3. Check the quality roadmap only for framework-wide prioritization and open only the contract that owns changed public behavior.
4. For a cross-deployable change, update or add the contract before implementing API and each
   affected browser-shell behavior.

Security vulnerabilities must follow [SECURITY.md](SECURITY.md) and must not be reported in a public issue.

## Development Setup

Required tool versions are documented in the root [README.md](README.md), `api/go.mod`, and `web/package.json`.

```bash
# API dependencies and generated dependency injection
cd api
go mod download
make wire

# Web dependencies
cd ../web
corepack pnpm install --frozen-lockfile
```

Use `api/docker-compose.yml` for the local PostgreSQL-backed API flow. The Web app can run with its bounded development mock BFF while you work on browser-only behavior.

## Change Boundaries

- Keep `api/` and `web/` independently deployable. They share HTTP contracts, not source code.
- Put reusable runtime infrastructure in core or capabilities, coherent business workflows in starters, and downstream product behavior outside the scaffold.
- Keep optional starters additive and disabled by default.
- Do not add a framework abstraction until it removes demonstrated complexity or matches an established local seam.
- Treat PostgreSQL as the only relational database compatibility target. Do not
  add SQLite code or tests; mock existing seams for unit tests and use
  disposable PostgreSQL for SQL behavior.
- Preserve stable `error_code`, `request_id`, pagination, authentication, and production mock-guard semantics.
- Update documentation, tests, and agent guidance when a public boundary changes.

## Verification

Run the smallest relevant checks while developing, then the root gate before submitting:

```bash
make agent-check # when agent guidance or skills changed
make check
```

`make check` includes governance; do not run both full commands back-to-back.

Useful focused commands:

```bash
cd api && go test ./...
cd web && corepack pnpm type-check
cd web && corepack pnpm lint
cd web && corepack pnpm test -- --run
cd web && corepack pnpm build
```

Container, dependency, SBOM, benchmark, and Compose commands are listed in [AGENTS.md](AGENTS.md). Run the checks whose contracts your change touches.

### Workspace Hygiene

Generated files never belong in commits. Run `make clean` to remove API
binaries, test executables, runtime logs, frontend build output, TypeScript
build metadata, and Python tool caches while retaining installed dependencies.
Run `make clean-all` only when a cold dependency reinstall is desired.

`node_modules/`, `.next/`, `dist/`, coverage output, and local `storage/logs/`
directories are disposable. A `.cache/` directory is also local-only, but may
contain reusable vulnerability-scanner downloads; keep those caches outside
the repository and do not remove them during ordinary cleanup.

## Pull Requests

A focused pull request should explain:

- the problem and reusable scaffold value;
- the semantic or contract decisions made;
- the verification evidence;
- security, compatibility, deployment, and rollback considerations;
- deliberate deferrals.

Keep generated files, formatting changes, and unrelated refactors out of the diff. Use conventional commit subjects such as `feat(api):`, `fix(web):`, `docs:`, `test:`, or `chore:`.

Branch and release behavior is defined in [docs/BRANCHING_AND_RELEASES.md](docs/BRANCHING_AND_RELEASES.md).

## License

By contributing, you agree that your contributions are licensed under the project [MIT License](LICENSE).
