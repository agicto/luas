---
name: downstream-app-extraction
description: Convert Luas into a downstream app safely. Use when deleting examples, replacing mock BFF routes, rebranding console surfaces, or checking product leakage.
---

# Downstream App Extraction

Separate scaffold behavior from product behavior while keeping inherited Luas
contracts, starters, deployables, and verification coherent.

## Repository Mode

Choose the mode from `pwd`, `git remote -v`, and
`git status --short --branch` before editing:

- **Scaffold mode**: the worktree is Luas or `origin` points at its source
  remote. Make only reusable scaffold changes. Product names, routes, jobs,
  content, credentials, and deployment behavior belong elsewhere.
- **Downstream mode**: the worktree is the derived application. Product
  behavior is allowed, but inherited surfaces still use Luas classifications.

If product work is requested in scaffold mode, stop before editing product
files and move to the downstream repository or request its path.

## Context Routing

Always read:

1. `CONTEXT.md` for canonical vocabulary.
2. `docs/SCAFFOLD_SURFACES.md` for the current catalog, downstream actions,
   and verification matrix.
3. Root `AGENTS.md` and the nearest deployable unit's `AGENTS.md`.

Read additional authorities only when their boundary is active:

- `contracts/README.md` when public HTTP behavior changes.
- `api/docs/ADDING_MODULE.md` for retained or added API starter behavior.
- `web/docs/ADDING_FEATURE.md` or `admin/docs/ADDING_FEATURE.md` for the
  downstream audience being implemented.
- `web/docs/MOCK_BFF.md`, `contracts/AUTHENTICATION.md`, and
  [references/mock-bff-auth.md](references/mock-bff-auth.md) when replacing
  mock routes or authentication adapters.
- `contracts/ASSETS.md`, `api/docs/ASSETS.md`, `web/docs/ASSETS.md`, and the
  asset reference below only when asset behavior is retained or removed.

For a retained optional starter, read only its matching reference:

| Starter | Conditional reference |
|---|---|
| `organization` | [references/organization.md](references/organization.md) |
| `permission` | [references/permission.md](references/permission.md) |
| `notification` | [references/notification.md](references/notification.md) |
| `asset` | [references/asset.md](references/asset.md) |
| `setting` | [references/setting.md](references/setting.md) |
| `usage` | [references/usage.md](references/usage.md) |

Do not load references for starters outside the requested downstream slice.

## Workflow

1. **Confirm the repository boundary.** Record the mode and, in downstream
   mode, the application name and product identifiers used for leakage checks.
2. **Inventory inherited surfaces.** List every changing API module, Web or
   Admin feature, mock route, console page, devtool, example, environment
   variable, background job, and deployment branch. Assign exactly one catalog
   classification to each.
3. **Choose keep, delete, replace, or rename.** Keep core and reusable
   capabilities unless the product deliberately swaps them. Keep or remove
   starters as complete workflows. Select retained API optional starters only
   through the canonical catalog and align `OPTIONAL_STARTERS`, migration and
   worker jobs, API replicas, and `NEXT_PUBLIC_OPTIONAL_FEATURES`. When deleting
   one, remove its catalog/provider contribution, owned contracts and
   migrations, browser feature, jobs, and environment selection together.
   Delete disposable examples and devtools; rename console surfaces only in
   downstream mode. Keep `web/`, `admin/`, or both according to the audiences
   the downstream product will maintain.
4. **Preserve contracts.** Update the owning contract first when behavior
   changes. Keep `error_code`, `request_id`, validation, pagination, and auth
   semantics aligned across independent deployables. Never replace documented
   HTTP contracts with cross-deployable source imports.
5. **Clean leakage.** Search for downstream and legacy product names,
   deployment and job names, remote URLs, demo credentials, routes, and content
   terms. Scaffold mode contains none except deliberate placeholders;
   downstream mode contains no user-visible Luas examples masquerading as
   product behavior.
6. **Verify the changed surfaces.** Use the nearest `AGENTS.md` and
   `docs/SCAFFOLD_SURFACES.md` matrix for focused proof. Use `make check` only
   for a genuinely cross-boundary extraction. In scaffold mode, run the helper
   below with the task's product identifiers before committing.

## Contamination Check

```bash
bash .agents/skills/downstream-app-extraction/scripts/check-downstream-contamination.sh \
  --expected-origin git@github.com:agicto/luas.git \
  --pattern "product-name" \
  --pattern "deployment-job-name"
```

The script intentionally contains no baked-in product names. Pass identifiers
from the current task.

## Completion Criteria

- Every changed surface has one catalog classification and an explicit action.
- Retained optional starters are selected consistently across replicas, jobs,
  migrations, and browser builds.
- Removed surfaces leave no owned contracts, routes, jobs, configuration, or
  user-visible scaffold behavior behind.
- Contract and deployable boundaries remain intact.
- Focused verification and the applicable contamination scan pass.

## Anti-patterns

- Editing product behavior while still in the Luas scaffold worktree.
- Treating devtools, examples, or mock BFF routes as production features.
- Removing contracts because shared source or a generic proxy seems faster.
- Rebranding source-scaffold documentation instead of the downstream app.
- Leaving demo credentials, old routes, product jobs, or product remotes.

## Related Skills

Navigation only; do not load these automatically:

- `domain-modeling` for a new canonical surface classification.
- `contract-evolution` when replacement changes public HTTP behavior.
