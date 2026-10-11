# Knowledge Base — Integration Steps

`plugin:add` already copied the API module, the migration, the contract, and the Web feature,
wired the `knowledge` starter into the API and the `knowledge` feature into the Web console through
slots, and merged the contract fragment when Node.js was available. Complete these steps in order;
each has a check that must pass.

1. **Merge the contract fragment if `plugin:add` could not.** Skip when
   `contracts/fragments/knowledge-base.yaml` no longer exists.
   `cd contracts && corepack pnpm install && corepack pnpm merge-fragment ../contracts/fragments/knowledge-base.yaml`
2. **Regenerate the browser contract types.** The Web feature binds its schemas to them, so Web type
   checks fail until this runs.
   `cd contracts && corepack pnpm generate && corepack pnpm check`
3. **Regenerate Wire.** `cd api && make wire`
4. **Enable the starter** in every environment that should serve it, next to `organization`:
   `OPTIONAL_STARTERS=organization,knowledge` (or `cd api && go run ./cmd/fastbuild starter:enable knowledge`).
5. **Enable the Web feature** in every Web environment that should show the Knowledge page, next to
   `organization`: `NEXT_PUBLIC_OPTIONAL_FEATURES=organization,knowledge`. It is a build-time value,
   so rebuild the Web after changing it. With the Go API adapter (`API_ADAPTER_ENABLED=true`) the
   browser routes forward to the API; in local development they use the mock BFF.
6. **Update generated records.** With a disposable PostgreSQL in `FASTBUILD_TEST_POSTGRES_DSN`:
   `cd api && make starter-catalog && FASTBUILD_UPDATE_GOLDEN_SCHEMA=1 go test ./database/migrations -run Golden`
7. **Verify.** `cd api && go test ./internal/modules/knowledge/... && make route-catalog-check`, then
   `cd contracts && corepack pnpm check:routes`, then
   `cd web && corepack pnpm type-check && corepack pnpm lint && corepack pnpm vitest run && corepack pnpm build`,
   then `make check` from the repository root.

The Web page is `/console/knowledge`: members list (newest first, cursor pages), search, and read
articles; owners and admins also create, edit (with `If-Match`), and delete them. Article Markdown
is shown as plain text; add a sanitizing Markdown renderer only if the product needs formatting.
There is no Admin Console view on purpose: operators do not read tenant content.

Ask the builder before changing who may write: the plugin limits writes to organization owners and
admins on purpose.
