---
name: plugin-integration
description: Add, update, or remove FastBuild registry plugins (features, themes, page blocks) when a requested capability may already exist; use business-implementation-planning when none fits.
---

# Plugin Integration

## Purpose

Turn "we need X" into a reviewed, verified capability by installing a registry plugin as source
instead of improvising the feature. The CLI performs the deterministic install; you perform the
plugin's adaptive integration steps; the repository's checks prove the result.

## When to Use

Use this skill when the builder asks for a capability, a visual theme, or a page section that a
registry plugin may provide, such as a knowledge base, tags, comments, a brand theme, or a pricing
table, phrased in any language; or asks to update or remove an installed plugin.

Skip it when the request is a product-specific workflow no plugin covers (plan it with
`business-implementation-planning`), or a change to code that is already installed (edit it as
ordinary project code; the project owns installed plugins).

## Workflow

Run commands from `api/` with `go run ./cmd/fastbuild` (or the installed `fastbuild` binary).

1. **Search.** `plugin:search <words> --format=json`. Search with the builder's own words and with
   synonyms in English and their language. Read `summary`, `use_when`, and `requires_starters`.
2. **Decide.** `plugin:info <name> --format=json`. Recommend the plugin only when a `use_when`
   situation matches the request. Tell the builder in one or two sentences what it adds, which
   starters it needs, and what it deliberately does not do. If nothing matches, say so and switch to
   `business-implementation-planning`; do not stretch a plugin to fit.
3. **Preview.** `plugin:add <name> --dry-run` lists every file it would write and every slot it would
   edit. A conflict means the project already has those files: stop and ask the builder rather than
   deleting anything.
4. **Install.** `plugin:add <name>`. It verifies checksums, rewrites imports to the project's Go
   module, inserts lines at named slots (`plugin:slots`), merges the contract fragment when Node.js
   is available, and records all of it in `fastbuild.plugins.lock`. Nothing is kept if it fails.
5. **Integrate.** Complete every step in the printed `INTEGRATION.md`, in order, and run the check
   each step names. Steps that need a decision, such as who may write, go to the builder.
6. **Verify.** `plugin:doctor` must pass, then the plugin's tests, the route and contract checks,
   and `make check` from the repository root. Report failures with their output; do not claim
   success without it.
7. **Hand over.** Summarize the installed plugin, the enabled starter, the integration steps done,
   any steps left to the builder (for example `OPTIONAL_STARTERS` in deployed environments), and
   that `plugin:status` shows later edits to the plugin's files.

Updating and removing:

- `plugin:update <name> --source=<newer release>` replaces untouched files and merges edited ones.
  Resolve every `<<<<<<< local` conflict and `*.fastbuild-new` file keeping the project's intent,
  rerun the affected integration steps, and finish only when `plugin:doctor` passes.
- `plugin:remove <name>` reverses files, slot lines, and contract additions; it refuses edited files
  unless the builder agrees to `--force`. Data stays in existing databases: say so.

## Rules

- Install only from the configured registry. Never copy plugin code from an unpinned source or
  edit `fastbuild.plugins.lock` or slot lines by hand; use the plugin commands.
- Do not change a plugin's authorization, data ownership, or contract during installation. Adapt
  it afterwards as a separate, reviewed change.
- Keep the plugin's tests. They are the project's guard for that capability.
- A plugin needs its required starters selected in every environment; say so explicitly.

## Verification

| Step | Proof |
|---|---|
| Install | `plugin:status` lists the plugin with every file `unchanged`; `plugin:doctor` passes |
| Contract | `cd contracts && corepack pnpm check && corepack pnpm check:routes` |
| Behavior | The plugin's package tests, with `FASTBUILD_TEST_POSTGRES_DSN` for PostgreSQL tests |
| Whole project | `make check` |

## Related Skills

- `business-implementation-planning` when no plugin fits.
- `contract-evolution` when an installed plugin's HTTP behavior must change.
- `verification-before-completion` when the right proof is unclear.
