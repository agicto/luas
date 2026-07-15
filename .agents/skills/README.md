# Luas Skill Index

This is the top-level index of all skills in the Luas scaffold. Codex CLI (and any tool following the `.agents/skills/` convention) auto-discovers skills from three roots based on the current working directory:

| When cwd is under | Skills loaded |
|---|---|
| Root (`/`) | this directory only |
| `api/` (any depth) | this directory + `api/.agents/skills/` |
| `web/` (any depth) | this directory + `web/.agents/skills/` |

Codex loads only **metadata** (name + description) into the system prompt at startup — the SKILL.md body is read on demand when the model decides to use a skill.

## Root Skills (apply everywhere)

| Skill | When to Use |
|---|---|
| [`contract-evolution`](./contract-evolution/) | Changing HTTP contracts across `contracts/`, `api/`, Web services, or mock BFF behavior |
| [`downstream-app-extraction`](./downstream-app-extraction/) | Converting Luas into a downstream app without leaking product behavior back into the scaffold |
| [`domain-modeling`](./domain-modeling/) | Resolving vocabulary, naming, or `starter` / `feature` / `capability` / `module` boundaries |
| [`grill-before-build`](./grill-before-build/) | Interview the user before underspecified or wide-impact changes |
| [`luas-code-review`](./luas-code-review/) | Reviewing a Luas diff against both standards and the originating request/spec |
| [`luas-framework-review`](./luas-framework-review/) | Review Luas as a global scaffold across security, performance, semantics, architecture, and AI workflows |
| [`systematic-debugging`](./systematic-debugging/) | Bug or flaky test with unclear cause — 4-phase reproduce → isolate → identify → verify |
| [`tdd-regression`](./tdd-regression/) | Fixing bugs, regressions, or contract drift with a failing test first |
| [`verification-before-completion`](./verification-before-completion/) | End-of-turn check that the change actually runs / tests pass / lint clean |
| [`pr-description-writer`](./pr-description-writer/) | Drafting a PR body or commit summary |

## API Skills (Go backend)

See [`../api/.agents/skills/`](../../api/.agents/skills/) — full table in [`api/AGENTS.md`](../../api/AGENTS.md#available-skills).

Quick map:

- **Structure**: `architecture-principles`, `module-creation`, `coding-standards`
- **Implementation**: `api-development`, `database-design`, `logging-standards`, `kest-flow`
- **Quality**: `testing-strategy`, `code-review-guide`
- **Operations**: `deployment`, `sql-migration-review`

## Web Skills (Next.js frontend)

See [`../web/.agents/skills/`](../../web/.agents/skills/) — full table in [`web/AGENTS.md`](../../web/AGENTS.md#ai-agent-skills).

Quick map:

- **Design**: `frontend-design`, `web-design-guidelines`, `ui-styling-guide`
- **Code patterns**: `vercel-react-best-practices`, `data-state-management`, `api-error-handling`
- **Infrastructure**: `environment-config`, `i18n-handler`, `utility-tooling`
- **Quality**: `webapp-testing`, `testing-standards`, `accessibility-audit`, `web-perf`
- **Strategy**: `project-strategy`, `skill-creator`

## How to Add a New Skill

1. Decide the scope: root (cross-cutting) → here; backend-specific → `api/.agents/skills/`; frontend-specific → `web/.agents/skills/`.
2. Run the `skill-creator` skill or copy the structure of `grill-before-build` for root skills.
3. Required: `SKILL.md` with YAML frontmatter (`name` + `description` ≤ 150 chars).
4. Optional: `scripts/` (executable helpers), `references/` (loaded on demand), `examples/`.
5. Add the new skill to the relevant table in this README and in the corresponding `AGENTS.md`.

## How to Verify Skills Are Loaded

```bash
# count discoverable skills (should be 36)
find . -maxdepth 5 -name "SKILL.md" -not -path "*/.template/*" | wc -l

# count what loads in api/ context (root + api = 21)
( cd api && find ../.agents/skills .agents/skills -name "SKILL.md" -not -path "*/.template/*" | wc -l )

# count what loads in web/ context (root + web = 25)
( cd web && find ../.agents/skills .agents/skills -name "SKILL.md" -not -path "*/.template/*" | wc -l )
```

In a codex session, ask: *"List the skills loaded in this session with their descriptions"* — the model should enumerate exactly the skills above.

## Conventions

- `luas-framework-review/scripts/check-vocabulary.sh` checks high-signal docs and every non-template `SKILL.md` for vocabulary drift from `CONTEXT.md`.
- `luas-framework-review/scripts/check-doc-links.py` checks local Markdown links across docs and agent guidance.
- `luas-framework-review/scripts/check-error-contracts.py` checks scaffold-level HTTP status and `error_code` alignment across contracts, API response constants, and Web fallbacks.
- `luas-framework-review/scripts/check-asset-boundary.py` checks private asset ownership, bounded object storage, ephemeral grants, inspection, cleanup, and API/Web contract alignment.
- `luas-framework-review/scripts/check-setting-boundary.py` checks finite typed definitions, strong version preconditions, reset history, cache/privacy behavior, audit minimization, and API/Web contract alignment.
- `luas-framework-review/scripts/check-surface-catalog.py` checks scaffold surface classification alignment across `CONTEXT.md`, `docs/SCAFFOLD_SURFACES.md`, and downstream extraction guidance.
- `luas-framework-review/scripts/check-branch-governance.sh` checks that branch/release docs stay aligned with CI-managed deployment branch mappings.
- **`name`** is `kebab-case`, ≤ 64 chars.
- **`description`** is the trigger condition, ≤ 150 chars in practice (codex hard limit is 1024 bytes). Lead with the *when*, not the *what*.
- **Directory name** matches the skill name. Skills starting with `.` or `_` are intentionally hidden from the loader (used for templates).
- **Body sections** follow: `Purpose / When to Use / Workflow / Anti-patterns / Pair With`.
- **`Pair With`** lets one skill point at adjacent skills so the model can chain them naturally.
