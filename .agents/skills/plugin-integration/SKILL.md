---
name: plugin-integration
description: Add a capability to a FastBuild project from the plugin registry (knowledge base, wiki, search, and other listed plugins). Use when a builder asks for a feature a plugin may already provide; use business-implementation-planning when no plugin fits.
---

# Plugin Integration

## Purpose

Turn "we need X" into a reviewed, verified capability by installing a registry plugin as source
instead of improvising the feature. The CLI performs the deterministic install; you perform the
plugin's adaptive integration steps; the repository's checks prove the result.

## When to Use

Use this skill when the builder asks for a capability that a registry plugin may provide, such as
a knowledge base, wiki, FAQ, or document search, phrased in any language.

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
3. **Preview.** `plugin:add <name> --dry-run` lists every file it would write. A conflict means the
   project already has those files: stop and ask the builder rather than deleting anything.
4. **Install.** `plugin:add <name>`. It verifies checksums, rewrites imports to the project's Go
   module, wires the starter, merges the contract fragment when Node.js is available, and records
   the plugin in `fastbuild.plugins.lock`. Nothing is kept if it fails.
5. **Integrate.** Complete every step in the printed `INTEGRATION.md`, in order, and run the check
   each step names. Steps that need a decision, such as who may write, go to the builder.
6. **Verify.** Run the plugin's tests, the route and contract checks, and `make check` from the
   repository root. Report anything that failed with its output; do not claim success without it.
7. **Hand over.** Summarize the installed plugin, the enabled starter, the integration steps done,
   any steps left to the builder (for example `OPTIONAL_STARTERS` in deployed environments), and
   that `plugin:status` shows later edits to the plugin's files.

## Rules

- Install only from the configured registry. Never copy plugin code from an unpinned source or
  edit `fastbuild.plugins.lock` by hand.
- Do not change a plugin's authorization, data ownership, or contract during installation. Adapt
  it afterwards as a separate, reviewed change.
- Keep the plugin's tests. They are the project's guard for that capability.
- A plugin needs its required starters selected in every environment; say so explicitly.

## Verification

| Step | Proof |
|---|---|
| Install | `plugin:status` lists the plugin with every file `unchanged` |
| Contract | `cd contracts && corepack pnpm check && corepack pnpm check:routes` |
| Behavior | The plugin's package tests, with `FASTBUILD_TEST_POSTGRES_DSN` for PostgreSQL tests |
| Whole project | `make check` |

## Related Skills

- `business-implementation-planning` when no plugin fits.
- `contract-evolution` when an installed plugin's HTTP behavior must change.
- `verification-before-completion` when the right proof is unclear.
