# Comments — Integration Steps

`plugin:add` already copied the module, the migration, and the contract, wired the `comments`
starter into the API, and merged the contract fragment when Node.js was available. Complete these
steps in order; each has a check that must pass.

1. **Merge the contract fragment if `plugin:add` could not.** Skip when
   `contracts/fragments/comments.yaml` no longer exists.
   `cd contracts && corepack pnpm install && corepack pnpm merge-fragment ../contracts/fragments/comments.yaml`
2. **Regenerate the browser contract types.**
   `cd contracts && corepack pnpm generate && corepack pnpm check`
3. **Regenerate Wire.** `cd api && make wire`. The comments provider asks for
   `domain.OrganizationRepository` and `domain.NotificationPublisher`, which the organization and
   notification starters always provide; `git diff api/internal/wiring/wire_gen.go` must show
   `comments.NewHandler`.
4. **Register the resource types members may comment on.** The plugin ships none, so until this
   step every request answers `422 COMMENTS.UNSUPPORTED_RESOURCE_TYPE`. Ask the builder which records
   get comments, then edit `ProvideResourceCatalog` in
   `api/internal/modules/comments/resources.go`: add one `ResourceType` per record with a checker
   that does one indexed, organization-scoped lookup through that module's existing seam (never a
   cross-module import of its internals) and returns `ResourceAccess{Read, Comment}`. A zero
   `ResourceAccess` means "not found", so members cannot probe other organizations' IDs. Add the
   reader the checker needs as a parameter; Wire passes it. Run `cd api && make wire` again and add
   a handler test with the new type.
5. **Enable the starter** in every environment that should serve it, next to `organization`:
   `OPTIONAL_STARTERS=organization,comments` (or `cd api && go run ./cmd/fastbuild starter:enable comments`).
   Add `notification` to deliver @mention notifications; without it mentions are stored silently.
6. **Update generated records.** With a disposable PostgreSQL in `FASTBUILD_TEST_POSTGRES_DSN`:
   `cd api && make starter-catalog && FASTBUILD_UPDATE_GOLDEN_SCHEMA=1 go test ./database/migrations -run Golden`
7. **Verify.** `cd api && go test ./internal/modules/comments/... && make route-catalog-check`, then
   `cd contracts && corepack pnpm check:routes`, then `make check` from the repository root.
8. **Optional UI.** To show a thread on a page, add a Web feature under `web/src/features/comments`
   that calls the routes in `contracts/COMMENTS.md` with the existing primitives, renders bodies as
   text (never raw HTML), and binds its schemas to the contract with `assertContract`. Follow
   `web/docs/ADDING_FEATURE.md`. Resolve mentioned user IDs to names through the organization
   members route.

Ask the builder before changing who may moderate: the plugin lets organization owners and admins
delete any comment, and never lets anyone but the author edit one.
