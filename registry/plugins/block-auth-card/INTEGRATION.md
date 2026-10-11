# Auth Card Block — Integration Steps

`plugin:add` copied `web/src/components/blocks/auth-card/index.tsx` and its test. Nothing is wired
automatically; complete these steps in order.

1. **Place it in the auth route group.** Use it in pages under `web/src/app/(auth)/`. Keep the
   existing forms in `web/src/features/auth/components/` as `children`: they own validation,
   submission, error mapping, and session handling. Do not move auth logic into the block.
2. **Check the outline.** If the auth layout already renders an `h1`, pass `headingLevel="h2"`.
3. **Pass translated copy.** Add the heading, description, divider label, footer link text, and legal
   text to the `auth` namespace in both `web/src/i18n/modules/auth/en-US.ts` and `zh-Hans.ts`; pass
   them from `getT('auth')` (Server Components) or `useT('auth')` (Client Components). The core i18n
   copy guard (`pnpm lint`) rejects hard-coded literals under `app/(auth)` and `components/blocks`.
4. **Alternative sign-in (optional).** Only add `alternativeActions` for providers the API actually
   supports; each button should start the provider flow defined by the auth adapter, not a mock.
5. **Verify.**
   `cd web && corepack pnpm vitest run src/components/blocks/auth-card src/test/auth-form-errors.test.tsx && corepack pnpm type-check && corepack pnpm lint`.
   Then sign in and out with the mock BFF (`make dev`) at 375px and desktop width in light and dark mode.
