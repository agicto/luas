# Adding A Web SPA Feature

## Placement

Use a feature-first directory:

```text
src/features/<feature>/
├── components/
├── hooks/
├── services/
├── store/
├── types.ts
└── index.ts
```

Create only the directories the feature owns. Route entries belong under
`src/routes/` and should import the feature's page component.

## Workflow

1. Read or update the owning contract under `../../contracts/`.
2. Add fixed endpoint paths and DTO validation in the feature service.
3. Add TanStack Query hooks for remote reads and mutations.
4. Add Zustand state only for genuinely shared browser UI state.
5. Add the route entry and keep it free of business logic.
6. Add English and Chinese i18next keys.
7. Add loading, empty, error, disabled, and responsive UI states.
8. Test services/hooks/components at their narrowest useful boundary.
9. Run the focused checks and production build.

## HTTP

- Call `src/http/client.ts`; do not call `fetch` from components.
- Default business responses use the standard Luas envelope.
- Use raw JSON mode only for an explicitly documented endpoint such as
  `/health/ready`.
- Validate security-sensitive or state-sensitive success data with Zod.
- Branch on backend `error_code`, not message text.
- Preserve `request_id` in support/diagnostic surfaces.
- Never accept a caller-controlled upstream URL or endpoint path.

## State

- TanStack Query owns API state, caching, retries, and invalidation.
- Zustand owns shared browser UI state.
- Component-local interaction stays in React state.
- Do not persist credentials, authenticated user records, permissions, or
  server-owned resource collections.
- Mutations are not retried automatically.

## Routing

TanStack Router generates `src/routeTree.gen.ts`. Never edit that file.
Represent search parameters with a schema and preserve typed links. Route
layouts own navigation composition; feature pages own workflow presentation.

## Security

Before adding authentication or a protected route, define the browser gateway
that owns HttpOnly cookies and CSRF/Origin enforcement. A client-side route
guard is never authorization.

Every `VITE_*` value is public. Add it to `.env.example`,
`src/vite-env.d.ts`, and `src/config/env.ts`, and document whether it is a
build-time value or a deployment routing value.

## Verification

```bash
corepack pnpm vitest run <test-file>
corepack pnpm type-check
corepack pnpm lint
corepack pnpm test -- --run
corepack pnpm build
```
