# AGENTS.md - Luas

Instructions for coding agents working in this monorepo.

## Repository Shape

Luas is an AI-era scaffold, not a product application:

- `api/`: Go backend (`github.com/zgiai/luas/api`) using Gin, Wire, GORM, and
  DDD-flavored starter modules.
- `web/`: customer-facing Next.js 16 application with React 19, TypeScript,
  Tailwind 4, shadcn, SSR, and server-owned browser adapters.
- `admin/`: project management console built with Vite 8, React 19, TanStack
  Router, TanStack Query, TypeScript, Tailwind 4, and shadcn/ui. It ships as
  static OSS/CDN assets.

All deployable units are independent and share HTTP contracts, never source
code. Downstream applications may deploy `web/`, `admin/`, or both.

## Fast Task Routing

Start with the smallest context that can answer the task:

1. Inspect the current diff/status and the nearest implementation and tests.
2. Read the nearest `api/AGENTS.md`, `web/AGENTS.md`, or
   `admin/AGENTS.md` only when editing that deployable unit.
3. Open `CONTEXT.md` for global vocabulary or ownership decisions, not for
   routine local edits.
4. Open the owning contract or architecture document only when that boundary
   is active.
5. Load at most one primary skill when its trigger clearly matches. Routine
   local work already covered by the nearest `AGENTS.md` and code may load none.
   A skill's `Pair With` section is navigation, not automatic chaining.

Do not preload document catalogs, all ADRs, both half guides, or all skills.
Expand context only when the current evidence leaves a real decision open.

### Skill Selection

Root skills live in `.agents/skills/`; API and Next.js Web add local skills.
The Admin Console uses root skills plus `admin/AGENTS.md`.
Codex loads skill metadata at discovery time and reads a full `SKILL.md` only
after selecting it.

- Use `luas-framework-review` only for an explicit framework-wide audit.
- Use `luas-code-review` for an explicit diff/PR review, not as an automatic
  post-edit ritual.
- Use `domain-modeling` only for global vocabulary or ownership boundaries,
  not ordinary symbol naming.
- Use `grill-before-build` only when a high-impact decision remains unresolved
  after local discovery.
- Use `verification-before-completion` only when the appropriate proof is
  genuinely unclear; routine tasks use the verification matrix below.

The complete skill index and helper catalog are in
[.agents/skills/README.md](.agents/skills/README.md).

## Authority Map

| Concern | Authority |
|---|---|
| Global vocabulary and surface ownership | [CONTEXT.md](CONTEXT.md) |
| Stable architecture and vertical flow | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| Branch and release behavior | [docs/BRANCHING_AND_RELEASES.md](docs/BRANCHING_AND_RELEASES.md) |
| CI and supply-chain policy | [docs/CI.md](docs/CI.md), [docs/DEPENDENCY_SECURITY.md](docs/DEPENDENCY_SECURITY.md), [docs/CONTAINER_SECURITY.md](docs/CONTAINER_SECURITY.md) |
| Shared HTTP envelope and compatibility | [contracts/README.md](contracts/README.md) |
| Capability-specific HTTP behavior | The owning file under `contracts/` |
| API implementation rules | [api/AGENTS.md](api/AGENTS.md) |
| Web implementation rules | [web/AGENTS.md](web/AGENTS.md) |
| Admin Console implementation rules | [admin/AGENTS.md](admin/AGENTS.md) |
| Long-running priorities | [docs/FRAMEWORK_QUALITY_ROADMAP.md](docs/FRAMEWORK_QUALITY_ROADMAP.md), [docs/STARTER_BUSINESS_ROADMAP.md](docs/STARTER_BUSINESS_ROADMAP.md) |

`CONTEXT.md` wins when terminology conflicts. The owning contract wins for
public HTTP behavior. The nearest `AGENTS.md` wins for local implementation.

## Repository Rules

1. API and browser shells never import each other's source. Browser clients
   talk to API behavior over HTTP.
2. Keep API imports under `github.com/zgiai/luas/api/...`; Web uses `@/...`.
3. Use `Luas` for the user-facing brand and `luas` for identifiers.
4. Never reintroduce the retired `LlamaFront`, `Hypership`, or `ZGO` brands.
5. Keep starter behavior product-neutral. Downstream product behavior belongs
   outside this scaffold.
6. Preserve the subtree-derived Git history; do not rewrite it.
7. Do not revert unrelated worktree changes.
8. PostgreSQL is the only relational database compatibility target. SQLite
   runtime code, drivers, dependencies, fixtures, migrations, and tests must
   remain absent. Unit tests use existing seams and test doubles; SQL behavior
   is verified against disposable PostgreSQL.

## Verification Budget

During implementation, run the narrowest check that proves the edited seam:

- Agent guidance or skill-only change: `make agent-check`
- API package change: targeted `go test`, then the relevant API tier
- Web feature change: targeted Vitest, type check, or lint for that surface
- Admin Console change: targeted Vitest, type check, lint, or static production build
- Contract change: owning contract guard plus tests on each changed side
- Release or genuinely cross-cutting change: `make check`

`make check` already includes `make governance`; never run both back-to-back.
Run the full gate once after focused checks pass, immediately before an
explicit release. An ordinary commit or push is not a release boundary. Do not
rerun an unchanged successful gate in the same turn.

## Common Commands

```bash
# Agent guidance
make agent-check
make governance

# API
cd api && make wire
cd api && go test ./internal/modules/<module>/...
cd api && bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 1 ./internal/modules/<module>/...
cd api && make route-catalog-check

# Web
cd web && corepack pnpm type-check
cd web && corepack pnpm lint
cd web && corepack pnpm vitest run <test-file>
cd web && corepack pnpm build

# Admin Console
cd admin && corepack pnpm type-check
cd admin && corepack pnpm lint
cd admin && corepack pnpm vitest run <test-file>
cd admin && corepack pnpm build

# Release gate
make check
```

Use the relevant half's docs for benchmarks, containers, migrations, security,
and operational checks rather than running every optional command by default.
