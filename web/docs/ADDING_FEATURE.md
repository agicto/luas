# Adding a Web Feature

Use this path for user-facing frontend behavior.

## Steps

1. Add or update the HTTP contract in `../../contracts/` when the feature calls the backend. Prefer `cd ../api && luas make:contract <name>` for a new cross-service feature.
2. Create `src/features/<name>/`.
3. Keep feature-owned code inside the feature folder:

```text
components/
hooks/
services/
server/
types.ts
index.ts
```

4. Put reusable app shell UI in `src/components/common/`.
5. Put shadcn-style primitives in `src/components/ui/`.
6. Compose the feature from `src/app/`; keep route files thin.
7. Run `pnpm type-check`, `pnpm lint`, and `pnpm test -- --run`, or `make check` from the repo root.

## Contract-First Workflow

Before implementing a service, inspect:

```bash
cd ../api && luas map --json
```

Use the matching `contracts/<name>.md` file as the source of truth for URLs, request bodies, response envelopes, pagination, and `error_code` values.

## Design Rules

- Feature services own HTTP calls and response mapping.
- Hooks own React Query or client state composition.
- Components receive typed props and avoid reaching into unrelated feature stores.
- Mock server behavior lives under `server/` until the API contract is backed by the Go service.
- Use existing semantic theme tokens and i18n helpers.

## Promotion Path

For vibe coding, start with mock data in `server/`, build the screen, then stabilize the contract before connecting to the backend. This keeps exploration fast while leaving a clear path to production code.
