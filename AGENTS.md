# AGENTS.md — Luas

Instructions for AI coding agents (Claude Code, Cursor, Windsurf, Copilot, etc.) working in this monorepo.

## What is Luas

Luas (Irish for *speed*) is a two-halves AI-era scaffold:

- **`api/`** — Go backend. Module: `github.com/zgiai/luas/api`. Gin + Wire DI + GORM, DDD-flavored modules, starter system (`user`, `apikey`, `audit`).
- **`web/`** — Next.js 16 / React 19 / TypeScript / Tailwind 4 / shadcn. Feature-first folders under `src/features/`.

The two halves are independent deployable units that share contracts, not code.

## Where to start

Each half has its own `AGENTS.md` with the detailed rules. Read those before editing:

- [api/AGENTS.md](api/AGENTS.md) — Go conventions, DDD layering, Wire DI, response patterns, testing
- [web/AGENTS.md](web/AGENTS.md) — Next.js patterns, feature folders, i18n, shadcn primitives

## Cross-cutting rules (apply everywhere)

1. **Don't cross the boundary.** API code never imports from `web/`. Web code only talks to API over HTTP. No shared codegen in this repo yet — keep contracts explicit.
2. **Preserve original module identity inside each half.** Inside `api/`, imports look like `github.com/zgiai/luas/api/internal/...`. Inside `web/`, imports look like `@/...`. Don't try to unify these.
3. **Brand strings.** The user-facing brand is **Luas** (capitalized). Lower-case `luas` only in identifiers (package names, binary names, env keys). Never re-introduce `LlamaFront`, `Hypership`, or `ZGO` into new code — those were the old names and have been deliberately cleaned up.
4. **History matters.** This repo was built by `git subtree add` from two upstreams; commits before the merge live under their original prefixes. Don't rewrite that history.

## Commands you might run

```bash
# api/
cd api && make wire && make run         # generate DI + start server
cd api && go vet ./...                  # quick correctness check
cd api && make test                     # run Go tests

# web/
cd web && pnpm install                  # install
cd web && pnpm dev                      # dev server with Turbopack
cd web && pnpm type-check               # TypeScript check
cd web && pnpm lint                     # ESLint
```

## When in doubt

Defer to the per-half AGENTS.md. They are the source of truth for layering, naming, and patterns. This top-level file is just the entry point.
