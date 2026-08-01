# Luas Skill Index

Luas keeps repeatable task workflows in `.agents/skills/`. Codex discovers
skill metadata first and loads a full `SKILL.md` only after selecting it.

## Discovery Scope

| Working directory | Repository skills available |
|---|---|
| Repository root | 10 root skills |
| `api/` | 10 root + 9 API skills |
| `web/` | 10 root + 13 Web skills |
| `admin/` | 10 root skills + local `AGENTS.md` |

The repository ships 32 skills in total. User, system, and plugin skills may
also appear in Codex; do not add a repository skill with the same name as a
known built-in skill.

`admin/` uses the root workflows and its focused local instructions. It does
not duplicate the Next-specific skill set under `web/.agents/skills/`.

## Routing Policy

1. Select at most one primary skill when its trigger clearly matches. Zero is
   valid for routine local work already covered by the nearest `AGENTS.md`.
2. Load another only when the task crosses its distinct ownership boundary.
3. Treat `Related Skills`/`Pair With` as navigation, not automatic chaining.
4. Keep mandatory everyday rules in the nearest `AGENTS.md`.
5. Keep detailed examples and variant references outside `SKILL.md` and load
   them only when needed.
6. Use focused checks while iterating. Run the full gate only for a
   cross-cutting change or explicit release; an ordinary push is not a release.

Avoid descriptions such as "use for all development" or "use when adding or
reviewing any UI." Descriptions should include positive triggers and important
negative boundaries so implicit selection remains precise.

## Root Skills

| Skill | Job |
|---|---|
| `contract-evolution` | Evolve shared HTTP behavior |
| `domain-modeling` | Resolve global vocabulary/ownership |
| `downstream-app-extraction` | Separate product behavior from the scaffold |
| `grill-before-build` | Resolve a genuinely blocking high-impact decision |
| `luas-code-review` | Review a concrete diff or PR |
| `luas-framework-review` | Run an explicit framework-wide audit |
| `systematic-debugging` | Isolate an unclear failure |
| `tdd-regression` | Fix a regression red/green |
| `verification-before-completion` | Resolve an unclear verification scope |
| `pr-description-writer` | Draft commit/PR communication |

## API Skills

| Skill | Job |
|---|---|
| `architecture-principles` | Decide API seams and structural ownership |
| `module-creation` | Create a route-owning starter module |
| `api-development` | Implement handlers, routes, and HTTP semantics |
| `database-design` | Design persistence and bounded queries |
| `logging-standards` | Design structured private telemetry |
| `testing-strategy` | Choose API test seams and doubles |
| `kest-flow` | Build running-API Markdown scenarios |
| `deployment` | Verify API image/runtime deployment |
| `sql-migration-review` | Review migration rollout safety |

Routine coding standards live in `api/AGENTS.md`; explicit diff review uses the
root `luas-code-review`. Separate duplicate skills for those concerns were
removed so API tasks do not load overlapping rulebooks.

## Web Skills

| Skill | Job |
|---|---|
| `frontend-design` | Establish substantial visual direction |
| `web-design-guidelines` | Review UI/UX quality |
| `ui-styling-guide` | Apply Luas tokens and primitives |
| `data-state-management` | Implement Query/Zustand state flow |
| `api-error-handling` | Implement client error semantics |
| `environment-config` | Change validated environment boundaries |
| `i18n-handler` | Change next-intl messages or routing |
| `utility-tooling` | Add a truly shared utility or hook |
| `testing-standards` | Write unit/component/integration tests |
| `webapp-testing` | Verify a running UI in a browser |
| `accessibility-audit` | Run a dedicated WCAG review |
| `web-perf` | Measure route and Web Vital performance |
| `vercel-react-best-practices` | Apply performance-sensitive React rules |

Codex provides the canonical `skill-creator`; Luas does not duplicate it inside
the Web scope.

## Skill Size And Metadata

- `name`: unique kebab-case, at most 64 characters.
- `description`: at most 1024 bytes and preferably at most 200 bytes.
- `SKILL.md`: at most 200 lines. Move optional tutorials and large examples to
  `references/` or `examples/`.
- Frontmatter: `name` and `description`; UI/policy metadata belongs in
  `agents/openai.yaml`.
- Detailed examples belong in `examples/` or `references/`.
- Deterministic repeated checks belong in `scripts/`.

Run:

```bash
make agent-check
bash .agents/skills/scripts/list-skills.sh
SKILL_VALIDATION_VERBOSE=1 bash .agents/skills/scripts/validate-skill.sh --all
```

`make agent-check` is the fast loop for agent guidance. `make governance`
executes all semantic/contract/supply-chain guards. `make check` already
includes governance and should be the single final release gate.

## Framework Guard Map

All guards live under `luas-framework-review/scripts/`.

| Changed surface | Guard |
|---|---|
| Vocabulary, English source, and local links | `check-vocabulary.sh`, `check-english-source.py`, `check-doc-links.py` |
| Shared errors and route discovery | `check-error-contracts.py`, `check-route-contract-discovery.py` |
| Auth/API keys/permissions | `check-auth-contract-boundary.py`, `check-api-key-boundary.py`, `check-permission-boundary.py` |
| Audit/notifications/assets/settings/usage/webhooks | `check-audit-boundary.py`, `check-notification-boundary.py`, `check-asset-boundary.py`, `check-setting-boundary.py`, `check-usage-boundary.py`, `check-webhook-boundary.py` |
| AI/email | `check-ai-boundary.py`, `check-email-boundary.py` |
| Rate limits/cache/PostgreSQL-only database policy/config/telemetry | `check-rate-limit-boundary.py`, `check-cache-boundary.py`, `check-database-boundary.py`, `check-config-authority.py`, `check-sensitive-telemetry.py` |
| Web performance/security/primitives | `check-web-performance-boundary.py`, `check-web-security-boundary.py`, `check-web-ui-primitive-boundary.py` |
| Admin Console architecture/security | `admin/scripts/check-architecture.mjs`, `check-auth-contract-boundary.py` |
| CI/dependencies/containers | `check-ci-actions.py`, `check-dependency-supply-chain.py`, `check-container-supply-chain.py` |
| Scaffold surfaces/starters | `check-surface-catalog.py`, `check-starter-catalog.py` |
| API imports and branch policy | `check-api-boundaries.sh`, `check-branch-governance.sh` |

Run only the owning guard during iteration. CI and the final repository gate
run the complete set.

## Adding Or Updating A Skill

Use the built-in `skill-creator`, then:

1. Put the skill at root, API, or Web scope according to its real owner.
2. Write concrete trigger and non-trigger examples.
3. Keep the core workflow concise; link optional resources conditionally.
4. Test representative prompts and any bundled scripts.
5. Run `make agent-check`.
6. Update this index and the nearest `AGENTS.md`.
