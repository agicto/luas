# AGENTS.md — Luas

Instructions for AI coding agents (Claude Code, Cursor, Windsurf, Copilot, etc.) working in this monorepo.

## What is Luas

Luas (Irish for _speed_) is a two-halves AI-era scaffold:

- **`api/`** — Go backend. Module: `github.com/zgiai/luas/api`. Gin + Wire DI + GORM, DDD-flavored modules, starter system (`user`, `apikey`, `audit`).
- **`web/`** — Next.js 16 / React 19 / TypeScript / Tailwind 4 / shadcn. Feature-first folders under `src/features/`.

The two halves are independent deployable units that share contracts, not code.

## Where to start

Read [CONTEXT.md](CONTEXT.md) first for the global Luas vocabulary. It is the source of truth for terms like `scaffold`, `starter`, `core`, `capability`, `feature`, `module`, `contract`, `mock BFF`, `console`, `error_code`, and `request_id`.

Each half has its own `AGENTS.md` with the detailed rules. Read those before editing:

- [api/AGENTS.md](api/AGENTS.md) — Go conventions, DDD layering, Wire DI, response patterns, testing
- [web/AGENTS.md](web/AGENTS.md) — Next.js patterns, feature folders, i18n, shadcn primitives

Workspace-level architecture docs:

- [CONTEXT.md](CONTEXT.md) — canonical global vocabulary for the whole scaffold
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — stable seams and vertical change flow
- [docs/BRANCHING_AND_RELEASES.md](docs/BRANCHING_AND_RELEASES.md) — branch roles, testing branches, release candidates, and deployment trigger rules
- [docs/FRAMEWORK_QUALITY_ROADMAP.md](docs/FRAMEWORK_QUALITY_ROADMAP.md) — long-running quality roadmap for professional, semantic, architecture-friendly iteration
- [docs/SKILL_GOVERNANCE_PLAN.md](docs/SKILL_GOVERNANCE_PLAN.md) — 30/60/90-day plan for keeping agent workflows aligned with vocabulary, contracts, and architecture
- [contracts/README.md](contracts/README.md) — HTTP contracts shared by `api/` and `web/`
- [api/docs/ADDING_MODULE.md](api/docs/ADDING_MODULE.md) — backend module checklist
- [web/docs/ADDING_FEATURE.md](web/docs/ADDING_FEATURE.md) — frontend feature checklist
- [web/docs/MOCK_BFF.md](web/docs/MOCK_BFF.md) — replacing or deleting the development mock BFF in downstream apps

## AI Agent Skills

The repo ships a Skills System at `.agents/skills/` (root) plus `api/.agents/skills/` and `web/.agents/skills/`. Codex CLI auto-discovers these based on cwd: the root + the matching half load when you cd into `api/` or `web/`. Each skill is a self-contained workflow loaded on demand when its description matches the task.

Root skills (apply everywhere):

| Skill | Use When |
|---|---|
| [`contract-evolution`](.agents/skills/contract-evolution/) | Changing HTTP contracts across `contracts/`, `api/`, Web services, or mock BFF behavior |
| [`downstream-app-extraction`](.agents/skills/downstream-app-extraction/) | Converting Luas into a downstream app without leaking product behavior back into the scaffold |
| [`domain-modeling`](.agents/skills/domain-modeling/) | Resolving vocabulary, naming, or `starter` / `feature` / `capability` / `module` boundaries |
| [`grill-before-build`](.agents/skills/grill-before-build/) | Request is underspecified or has wide impact (persistence, permissions, deployment, user workflows) |
| [`luas-code-review`](.agents/skills/luas-code-review/) | Reviewing a Luas diff against both standards and the originating request/spec |
| [`luas-framework-review`](.agents/skills/luas-framework-review/) | Reviewing Luas as a global scaffold for security, performance, semantics, architecture, or AI workflows |
| [`systematic-debugging`](.agents/skills/systematic-debugging/) | Bug or flaky test where the cause is not obvious |
| [`tdd-regression`](.agents/skills/tdd-regression/) | Fixing bugs, regressions, or contract drift with a failing test first |
| [`verification-before-completion`](.agents/skills/verification-before-completion/) | End-of-turn check that what you built actually runs / tests / lints |
| [`pr-description-writer`](.agents/skills/pr-description-writer/) | Drafting a PR body or commit summary |

Backend-specific skills are listed in [api/AGENTS.md](api/AGENTS.md#available-skills); frontend-specific in [web/AGENTS.md](web/AGENTS.md#ai-agent-skills). Full index at [.agents/skills/README.md](.agents/skills/README.md).

Helper scripts shipped with skills:

- `.agents/skills/verification-before-completion/scripts/run-tiers.sh <0|1|2> [scope...]` — auto-detects api/ vs web/ and runs the chosen tier.
- `.agents/skills/luas-framework-review/scripts/scaffold-architecture-report.py` — generate an optional HTML architecture review report in `$TMPDIR`.
- `.agents/skills/luas-framework-review/scripts/check-doc-links.py` — verify local Markdown links across docs and agent guidance.
- `.agents/skills/luas-framework-review/scripts/check-error-contracts.py` — verify scaffold-level HTTP status and `error_code` alignment across contracts, API, and Web.
- `.agents/skills/luas-framework-review/scripts/check-branch-governance.sh` — verify branch/release docs match CI-managed deployment branch mappings.
- `.agents/skills/pr-description-writer/scripts/scaffold-pr-body.sh [base]` — generate a PR body draft from `git log` + `git diff`.
- `api/.agents/skills/sql-migration-review/scripts/check-migration.sh <file>` — static checks for migration files.

CI enforces the canonical references via [.github/workflows/skill-self-test.yml](.github/workflows/skill-self-test.yml) and [.github/workflows/ci.yml](.github/workflows/ci.yml).

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

# repo root
make check                              # API tests + web type/lint/test/build
```

## When in doubt

Defer to the per-half AGENTS.md. They are the source of truth for layering, naming, and patterns. This top-level file is just the entry point.
