# Settings Page Block — Integration Steps

`plugin:add` copied `web/src/components/blocks/settings-page/index.tsx` and its test. Nothing is
wired automatically; complete these steps in order.

1. **Place it.** Use it as the body of a settings route, for example under
   `web/src/app/(protected)/(console)/console/settings/`. The console layout already renders the
   page shell; pass `headingLevel="h2"` if the layout or page already renders an `h1`.
2. **Keep forms in their features.** Each section's `content` is an existing feature component (for
   example the setting panels in `web/src/features/setting/components/`). The block does not submit,
   validate, or fetch; the forms keep their own TanStack Query mutations and error handling.
3. **Pass translated copy.** Add the page title, description, navigation label, and section titles to
   the owning namespace (for example `settings`) in both `web/src/i18n/modules/<namespace>/en-US.ts`
   and `zh-Hans.ts`; pass them from `getT('<namespace>')` or `useT('<namespace>')`.
4. **Section ids.** Use stable, URL-friendly ids; they are the anchor targets. To highlight the
   section in view, pass `activeSectionId` from the route or a small client wrapper.
5. **Verify.**
   `cd web && corepack pnpm vitest run src/components/blocks/settings-page && corepack pnpm type-check && corepack pnpm lint`.
   Then check at 375px that the navigation scrolls horizontally and each link jumps to its section,
   in light and dark mode (`make dev`).
