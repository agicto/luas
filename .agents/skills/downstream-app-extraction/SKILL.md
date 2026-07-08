---
name: downstream-app-extraction
description: Convert Luas into a downstream app safely. Use when deleting examples, replacing mock BFF routes, rebranding console surfaces, or checking product leakage.
---

# Downstream App Extraction

## Purpose

Turn Luas into a downstream app without confusing scaffold code with product code. Use this skill to decide what to keep, delete, replace, rename, or guard when a team starts from Luas.

## Operating Modes

Pick one mode before editing:

- **Scaffold mode**: cwd is the Luas repository or `origin` points at the Luas remote. Only make scaffold-neutral changes: docs, skills, contracts, starters, examples, guardrails, or reusable defaults. Do not add downstream product names, routes, jobs, content, credentials, or deployment behavior.
- **Downstream mode**: cwd is a downstream app repository. Product-specific behavior is allowed, but keep Luas vocabulary while classifying what was inherited from the scaffold.

If a user asks for product behavior while the worktree is still Luas scaffold mode, stop before editing product files and move to the downstream repository or ask for the correct path.

## Source Material

Read these before extraction work:

1. `CONTEXT.md` for `scaffold`, `downstream app`, `starter`, `mock BFF`, `console`, `devtools`, and `example`.
2. `AGENTS.md`, plus `api/AGENTS.md` or `web/AGENTS.md` when touching a half.
3. `web/docs/MOCK_BFF.md` when deleting or replacing mock route handlers.
4. `api/docs/ADDING_MODULE.md` when keeping or adding backend starter-style behavior.
5. `web/docs/ADDING_FEATURE.md` when keeping or adding Web feature behavior.
6. `contracts/README.md` when real API behavior or HTTP client behavior changes.

## Surface Classification

Classify each touched surface before changing it.

| Surface | Keep In Luas? | Downstream Action |
|---|---|---|
| `core` | yes | keep unless the downstream app intentionally swaps infrastructure |
| default starter | yes | keep, remove, or rename by product need |
| optional starter | yes | wire in only when the product needs it |
| capability | yes | keep if reusable; configure behind product-owned settings |
| mock BFF | yes, development-only | replace with real API, delete, or keep local-only with guards |
| console | yes, replaceable | rename or redesign for product workspace needs |
| devtools | yes, internal only | delete from production app unless explicitly needed |
| example | yes, isolated | delete or replace with product feature |
| product-specific behavior | no | belongs only in downstream mode |

## Workflow

1. **Confirm repository boundary**
   - Run `pwd`, `git remote -v`, and `git status --short --branch`.
   - In scaffold mode, reject product-specific edits and keep the change reusable.
   - In downstream mode, record the downstream app name and expected product identifiers for leakage checks.

2. **Inventory inherited surfaces**
   - List API modules, Web features, mock BFF routes, console pages, devtools, examples, env vars, background jobs, and deployment branches that will change.
   - Mark each item with one classification from the table above.

3. **Choose keep/delete/replace**
   - Keep core and capabilities unless there is a clear product reason.
   - Keep default starters when they remain useful as business-ready building blocks.
   - Delete examples and devtools when they no longer teach or support the downstream app.
   - Replace mock BFF routes with the real API path or same-origin proxy for production.
   - Rename console surfaces only in downstream mode.

4. **Preserve contracts**
   - Update `contracts/README.md` first if request or response behavior changes.
   - Keep `error_code`, `request_id`, validation errors, and pagination stable across API, Web services, and any retained mock BFF.
   - Do not share source between `api/` and `web/`; share documented contracts.

5. **Clean product leakage**
   - Search for downstream product names, old product names, deployment names, job names, remote URLs, demo credentials, and content pipeline terms.
   - In scaffold mode, product identifiers must not appear in committed files unless they are deliberately documented as placeholders.
   - In downstream mode, Luas scaffold examples should not remain as user-visible product behavior.

6. **Verify**
   - Run the narrow commands for changed surfaces and the broader tier that proves the extraction still works.
   - For mock BFF replacement, run Web type/lint/test/build and a browser or curl check for auth, CORS, credentials, and error envelopes.
   - Before committing in Luas scaffold mode, run the product leakage script with the relevant identifiers.

## Helper Script

Use `scripts/check-downstream-contamination.sh` to scan the current repository for product-specific leakage and to confirm the expected origin remote:

```bash
bash .agents/skills/downstream-app-extraction/scripts/check-downstream-contamination.sh \
  --expected-origin git@github.com:zgiai/luas.git \
  --pattern "product-name" \
  --pattern "deployment-job-name"
```

Pass the product-specific identifiers from the current task. The script intentionally has no baked-in product names.

## Verification Matrix

| Change | Verification |
|---|---|
| Scaffold-neutral docs or skills | `validate-skill.sh --all`, vocabulary check, `git diff --check` |
| API starter kept or renamed | `cd api && go test ./...` or targeted module tests |
| Web feature kept or renamed | `cd web && pnpm type-check && pnpm lint && pnpm test -- --run` |
| Mock BFF deleted or replaced | `cd web && pnpm vitest run src/test/mock-bff-route-contract.test.ts`, then adapt/delete that guard when no mock routes remain |
| Cross-boundary extraction | `make check` plus targeted searches for product and scaffold leftovers |
| Scaffold-mode commit | contamination script with expected origin and task-specific product identifiers |

## Anti-patterns

- Editing product behavior while the worktree still points at the Luas scaffold remote.
- Treating `devtools` or examples as product features.
- Shipping mock BFF routes as a production API by accident.
- Rebranding Luas scaffold docs in the source scaffold instead of in a downstream app.
- Deleting contracts because codegen or shared source feels faster.
- Leaving demo credentials, old routes, content jobs, or product remotes in Luas.

## Pair With

- `luas-framework-review` when choosing the next scaffold governance slice.
- `domain-modeling` when a surface needs a new canonical classification.
- `contract-evolution` when replacing mock BFF behavior changes HTTP contracts.
- `luas-code-review` before merging extraction diffs.
- `verification-before-completion` before reporting extraction complete.
