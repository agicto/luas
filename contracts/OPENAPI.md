# OpenAPI Contract

[`openapi.yaml`](openapi.yaml) is the OpenAPI 3.1 machine contract for every Luas HTTP operation,
including every optional starter. Only `GET /` (service banner) and `GET /metrics` (Prometheus text)
are intentionally undocumented. The owning Markdown contract continues to define authorization,
privacy, idempotency, retention, and other semantics that OpenAPI cannot express precisely.

## Toolchain

- Redocly CLI validates OpenAPI structure and repository rules.
- `openapi-typescript` generates deterministic TypeScript types independently into `web/` and
  `admin/`; neither browser shell imports source from the other.
- The route check assembles the real Go route catalog with every optional starter selected and
  infrastructure disabled. It fails when a described operation has no route and when a route has
  no described operation, so a new endpoint cannot ship without its contract.
- `oasdiff` compares a pull request against its target commit and rejects unreviewed breaking
  changes.

Install and run from the repository root:

```bash
cd contracts
corepack pnpm install --frozen-lockfile
corepack pnpm check
corepack pnpm check:routes
corepack pnpm generate
```

Generated files are committed so downstream projects and editors receive types without running a
generator during installation. CI fails when either copy is stale.

## Change Workflow

1. Update the owning Markdown contract and `openapi.yaml` together. A new route fails
   `corepack pnpm check:routes` until it is described.
2. Add only behavior that the API actually implements.
3. Run `cd contracts && corepack pnpm generate`.
4. Bind browser request, response, and error types to generated schemas at the owning feature seam.
5. Keep runtime validation for untrusted JSON; generated TypeScript types are not validators.
6. Run contract, API, mock BFF, and browser tests for the changed operation.

Breaking changes require an explicit versioning and migration decision. Do not weaken the CI diff
gate to make a breaking change pass. Additive endpoints and optional response fields are normally
compatible, while removing operations, narrowing accepted input, or adding required fields may not
be.
