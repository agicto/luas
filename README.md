# Luas

> **Luas** — Irish for *speed*. An AI-era full-stack scaffold: opinionated Go API + Next.js web, designed to ship apps fast on stable rails.

```
luas/
├── api/   # Go backend — Gin + Wire + GORM, DDD modules, AI-capability ready
├── web/   # Next.js 16 + React 19 + Tailwind 4 + shadcn, AI-agent friendly
└── ...
```

## Why Luas

| | |
|---|---|
| **Speed** | "Luas" literally means *speed* in Irish. Sensible defaults, no boilerplate ceremony, opinionated wiring. |
| **Stable rails** | Both halves are battle-tested and conservative: no exotic dependencies, no half-finished abstractions. |
| **Great patterns** | DDD-flavored modules on the API side, feature-first folders on the web side, AGENTS.md on both. |
| **Architecture-first** | The two services share contracts, not code. Cleanly deployable as separate units. |

## Quick start

### API (`api/`)

```bash
cd api
cp .env.example .env
make wire     # generate DI
make run      # start server on 127.0.0.1:8025
```

See [api/README.md](api/README.md) for the full Go backend guide.

### Web (`web/`)

```bash
cd web
pnpm install
pnpm dev      # start Next.js on :3000
```

See [web/README.md](web/README.md) for the full frontend guide.

## Architecture and contracts

- [CONTEXT.md](CONTEXT.md) — canonical Luas vocabulary for humans and agents.
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — deployable-unit boundaries and vertical change flow.
- [docs/CI.md](docs/CI.md) — workflow roles, runner compatibility, immutable action pins, and update procedure.
- [contracts/README.md](contracts/README.md) — shared HTTP response, error, pagination, and request ID contracts.
- [contracts/ORGANIZATIONS.md](contracts/ORGANIZATIONS.md) — additive optional organization ownership foundation and its explicit deferrals.
- [docs/SCAFFOLD_SURFACES.md](docs/SCAFFOLD_SURFACES.md) — what to keep, delete, or replace when turning Luas into a downstream app.
- [docs/FRAMEWORK_QUALITY_ROADMAP.md](docs/FRAMEWORK_QUALITY_ROADMAP.md) — long-running quality roadmap across semantics, security, performance, usability, and AI workflows.
- [docs/STARTER_BUSINESS_ROADMAP.md](docs/STARTER_BUSINESS_ROADMAP.md) — ready-to-use starter review and prioritized reusable business starter roadmap.
- [docs/SKILL_GOVERNANCE_PLAN.md](docs/SKILL_GOVERNANCE_PLAN.md) — 30/60/90-day plan for skill governance and AI workflow quality.

## Working with AI agents

Both halves were designed for AI-assisted development. Start with the global [CONTEXT.md](CONTEXT.md) for the shared Luas vocabulary, then use the top-level [AGENTS.md](AGENTS.md) plus the per-half [api/AGENTS.md](api/AGENTS.md) and [web/AGENTS.md](web/AGENTS.md) for navigation, boundaries, and conventions.

## Verification

```bash
make governance  # root semantic, contract, docs, CI, surface, branch, package, and skill guardrails
make check       # governance + API tier 1 + Web tier 2
```

## History

This repo merges two previous projects, with full commit history preserved:

- `api/` — formerly [`zgiai/zgo`](https://github.com/zgiai/zgo)
- `web/` — formerly [`zgiai/zweb`](https://github.com/zgiai/zweb) (previously branded *LlamaFront* / *Hypership Web Console*)

Module / package identifiers have been renamed to match the new brand:

- Go module: `github.com/zgiai/zgo` → `github.com/zgiai/luas/api`
- Web package: `llamafront-ai-scaffold` → `luas-web`

## License

MIT (inherited from both source repos).
