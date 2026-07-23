# Luas API Skills

API skills provide task-specific workflows beyond the mandatory rules in
`api/AGENTS.md`. Codex reads only metadata until a skill is selected.

## Skills

| Skill | Select for |
|---|---|
| `architecture-principles` | Seams, interfaces, starter/capability classification |
| `module-creation` | New route-owning default or optional starter |
| `api-development` | Routes, handlers, validation, pagination, HTTP responses |
| `database-design` | Persistence models, indexes, bounded query design |
| `logging-standards` | Structured event and redaction decisions |
| `testing-strategy` | Test seam, double, and integration strategy |
| `kest-flow` | Markdown flows against a running API |
| `deployment` | API image and runtime deployment checks |
| `sql-migration-review` | Migration compatibility, locks, and rollback |

Routine code standards stay in `api/AGENTS.md`; concrete diff review uses root
`luas-code-review`. This avoids loading overlapping standards and review
workflows during ordinary API implementation.

## Rules

- Select one primary skill.
- Read examples only when nearby production code is insufficient.
- Keep `SKILL.md` below 500 lines.
- Put deterministic checks in `scripts/`.
- Keep frontmatter descriptions precise enough to exclude routine unrelated
  work.
- Run focused package tests while iterating; run root `make check` once at the
  release boundary.

## Commands

```bash
# From the repository root
bash .agents/skills/scripts/list-skills.sh
make agent-check

# From api/
go test ./internal/modules/<module>/...
bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 1 ./internal/modules/<module>/...
```

The root index at [`../../../.agents/skills/README.md`](../../../.agents/skills/README.md)
defines cross-repository routing and the governance guard map.
