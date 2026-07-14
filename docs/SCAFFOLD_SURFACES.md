# Scaffold Surfaces

This catalog classifies the replaceable and reusable surfaces that ship with Luas.
Use it when turning Luas into a downstream app, adding a starter, deleting an example,
or deciding whether behavior belongs in the source scaffold.

The canonical vocabulary still lives in [`../CONTEXT.md`](../CONTEXT.md). This file
applies that vocabulary to the current repository tree and points each surface at
the verification that proves it is still safe to keep, delete, or replace.

## Surface Catalog

| Surface | Current Luas homes | Lifecycle in Luas | Downstream action | Verification |
|---|---|---|---|---|
| `core` | `api/internal/bootstrap`, `api/internal/infra`, `api/pkg`, `web/src/components/ui`, `web/src/config`, `web/src/http`, `web/src/i18n`, `web/src/themes` | Long-lived runtime and infrastructure. | Keep unless the downstream app intentionally swaps infrastructure. | `make check`; `bash .agents/skills/luas-framework-review/scripts/check-api-boundaries.sh` |
| `default starter` | `api/internal/domain`, `api/internal/modules/user`, `api/internal/modules/apikey`, `api/internal/modules/audit`, `web/src/features/auth` | Business-ready starter behavior wired into the default scaffold. | Keep, remove, or rename by product need while preserving contracts. | `cd api && make test`; `cd web && pnpm test -- --run`; `make check` |
| `optional starter` | `api/internal/modules/organization` is the first backend foundation; available starters are cataloged by `api/internal/starter`, disabled by default, and selected with `OPTIONAL_STARTERS`. | Starter-quality behavior kept outside default runtime assembly until explicitly chosen. | Select only what the product needs, keep migration jobs and all replicas on the same selection, and remove both the catalog entry and owned surfaces when deleting it. | `cd api && make test`; `go run ./cmd/luas route:list`; `make governance` |
| `capability` | `api/internal/capabilities`, reusable helpers under `api/pkg`, technical Web helpers without product workflow ownership | Reusable technical integration or helper. | Keep when reusable; configure behind product-owned settings. | Targeted package tests; `make check` |
| `mock BFF` | `web/src/app/api`, `web/src/app/api/_shared`, `web/src/features/*/server` mock state | Development-only browser-contract substitute. | Replace with production endpoints or an explicit adapter, delete, or keep local-only with production and same-origin mutation guards. | `cd web && pnpm vitest run src/test/mock-bff-route-contract.test.ts`; `python3 .agents/skills/luas-framework-review/scripts/check-error-contracts.py` |
| `console` | `web/src/app/(protected)/(console)` and shared console UI under `web/src/components/features/console` | Replaceable authenticated scaffold workspace. | Rename or redesign in downstream mode; do not turn it into a fixed downstream workspace in Luas. | `cd web && pnpm type-check && pnpm lint && pnpm test -- --run` |
| `devtools` | `web/src/app/(protected)/(devtools)`, styleguide and i18n test routes | Internal playground or demonstration surface. | Delete from production apps unless explicitly needed by the product team. | `cd web && pnpm build`; downstream route/product leakage scan |
| `example` | `web/src/features/example`, `web/src/app/api/example`, example docs under `.agents/skills/**/examples` | Disposable teaching or scaffold demonstration code. | Delete or replace with product features. | `cd web && pnpm vitest run src/test/mock-bff-route-contract.test.ts`; doc link check |
| `product-specific behavior` | Not allowed in the Luas source scaffold. | Belongs only in downstream app mode. | Move to the downstream repository or keep it out of the Luas commit. | `bash .agents/skills/downstream-app-extraction/scripts/check-downstream-contamination.sh --expected-origin git@github.com:zgiai/luas.git --pattern "<product-identifier>"` |

## Localization Ownership

Formal Web copy in the public site, auth starter, replaceable console, root metadata,
and their shared shell components is part of the governed scaffold experience. It must
use the i18n message tree and pass `cd web && pnpm lint:i18n-copy`. Exact brand names,
technical identifiers, and `<code>` content are narrow literal exceptions. `devtools`
and `example` remain intentionally disposable and are excluded from this core-copy
guard; the dependency-light `global-error.tsx` fallback is also excluded so it can
render when the normal root runtime fails.

## Downstream Extraction Rules

1. Start from repository identity: run `pwd`, `git remote -v`, and `git status --short --branch`.
2. Classify each touched file with exactly one surface from the catalog before editing.
3. Update [`../contracts/README.md`](../contracts/README.md) first when a replacement changes HTTP behavior.
4. Use [`../web/docs/MOCK_BFF.md`](../web/docs/MOCK_BFF.md) when deleting, replacing, or keeping mock BFF routes.
5. Use [`../api/docs/ADDING_MODULE.md`](../api/docs/ADDING_MODULE.md) for starter-style backend behavior.
6. Select retained optional API starters with one `OPTIONAL_STARTERS` value shared by replicas and database jobs; do not hand-register their routes or migrations.
7. Use [`../web/docs/ADDING_FEATURE.md`](../web/docs/ADDING_FEATURE.md) for product-facing Web features.
8. Run the contamination scan before committing scaffold-mode work that touched examples, devtools, console surfaces, mock BFF behavior, product copy, deployment names, or downstream docs.

## Verification Matrix

| Change | Required checks |
|---|---|
| Surface classification or this catalog | `python3 .agents/skills/luas-framework-review/scripts/check-surface-catalog.py`; vocabulary check; doc link check |
| Mock BFF kept, replaced, or deleted | `cd web && pnpm vitest run src/test/mock-bff-route-contract.test.ts`; update or remove the guard when no mock routes remain |
| API starter kept, removed, or added | `cd api && make test`; `python3 .agents/skills/luas-framework-review/scripts/check-starter-catalog.py`; package boundary check |
| Web feature, console, devtools, or example changed | `cd web && pnpm type-check && pnpm lint && pnpm test -- --run` |
| Cross-boundary downstream extraction | `make check`; error contract check; contamination scan |
