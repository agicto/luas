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
make run      # start server on :8025
```

See [api/README.md](api/README.md) for the full Go backend guide.

### Web (`web/`)

```bash
cd web
pnpm install
pnpm dev      # start Next.js on :3000
```

See [web/README.md](web/README.md) for the full frontend guide.

## Working with AI agents

Both halves were designed for AI-assisted development. The top-level [AGENTS.md](AGENTS.md) plus the per-half [api/AGENTS.md](api/AGENTS.md) and [web/AGENTS.md](web/AGENTS.md) tell coding agents (Claude Code, Cursor, Windsurf, Copilot, etc.) how to navigate, where the boundaries are, and which conventions to follow.

## History

This repo merges two previous projects, with full commit history preserved:

- `api/` — formerly [`zgiai/zgo`](https://github.com/zgiai/zgo)
- `web/` — formerly [`zgiai/zweb`](https://github.com/zgiai/zweb) (previously branded *LlamaFront* / *Hypership Web Console*)

Module / package identifiers have been renamed to match the new brand:

- Go module: `github.com/zgiai/zgo` → `github.com/zgiai/luas/api`
- Web package: `llamafront-ai-scaffold` → `luas-web`

## License

MIT (inherited from both source repos).
