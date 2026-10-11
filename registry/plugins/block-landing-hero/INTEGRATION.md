# Landing Hero Block — Integration Steps

`plugin:add` copied `web/src/components/blocks/landing-hero/index.tsx` and its test. Nothing is wired
automatically; complete these steps in order.

1. **Place it.** Use it on a public page under `web/src/app/(site)/`, typically replacing the hero and
   feature sections of `page.tsx`. Keep exactly one `h1` on the page: the block renders it.
2. **Pass translated copy.** Add the eyebrow, heading, description, call-to-action labels, feature
   heading, and feature titles and descriptions to the `site` namespace (or the owning one) in both
   `web/src/i18n/modules/<namespace>/en-US.ts` and `zh-Hans.ts`; pass them from `getT('<namespace>')`.
   Brand names stay literals only where the core i18n copy guard allows them.
3. **Media.** Pass `next/image` with a translated `alt` (or `alt=""` when purely decorative) and
   explicit `width`/`height` or `fill` with `sizes`, so the hero does not shift layout.
4. **Calls to action.** Use `Button asChild` around `Link` to real routes from
   `web/src/constants/routes.ts`; keep the primary action first.
5. **Verify.**
   `cd web && corepack pnpm vitest run src/components/blocks/landing-hero && corepack pnpm type-check && corepack pnpm lint`.
   Then check at 375px, tablet, and desktop width in light and dark mode (`make dev`), and run
   `cd web && corepack pnpm build` to keep the route bundle budget green.
