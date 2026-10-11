# Knowledge Base — Integration Steps

`plugin:add` already copied the module, the migration, and the contract, wired the `knowledge`
starter into the API, and merged the contract fragment when Node.js was available. Complete these
steps in order; each has a check that must pass.

1. **Merge the contract fragment if `plugin:add` could not.** Skip when
   `contracts/fragments/knowledge-base.yaml` no longer exists.
   `cd contracts && corepack pnpm install && corepack pnpm merge-fragment ../contracts/fragments/knowledge-base.yaml`
2. **Regenerate the browser contract types.**
   `cd contracts && corepack pnpm generate && corepack pnpm check`
3. **Regenerate Wire.** `cd api && make wire`
4. **Enable the starter** in every environment that should serve it, next to `organization`:
   `OPTIONAL_STARTERS=organization,knowledge` (or `cd api && go run ./cmd/fastbuild starter:enable knowledge`).
5. **Update generated records.** With a disposable PostgreSQL in `FASTBUILD_TEST_POSTGRES_DSN`:
   `cd api && make starter-catalog && FASTBUILD_UPDATE_GOLDEN_SCHEMA=1 go test ./database/migrations -run Golden`
6. **Verify.** `cd api && go test ./internal/modules/knowledge/... && make route-catalog-check`, then
   `cd contracts && corepack pnpm check:routes`, then `make check` from the repository root.
7. **Optional UI.** To give members a page, add a Web feature under `web/src/features/knowledge`
   that calls the routes in `contracts/KNOWLEDGE.md` (list, read, search) with the existing
   primitives, and bind its schemas to the contract with `assertContract`. Follow
   `web/docs/ADDING_FEATURE.md`.

Ask the builder before changing who may write: the plugin limits writes to organization owners and
admins on purpose.
