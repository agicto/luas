# Pricing Table Block — Integration Steps

`plugin:add` copied `web/src/components/blocks/pricing-table/index.tsx` and its test. Nothing is
wired automatically; complete these steps in order.

1. **Place it.** Usually on a public page under `web/src/app/(site)/`, for example a new
   `pricing/page.tsx`, or on an upgrade page inside the console. Add a route constant in
   `web/src/constants/routes.ts` if you link to it from navigation.
2. **Pass translated copy and prices.** Add plan names, descriptions, feature labels, the
   included/not-included labels, badge text, and call-to-action labels to the owning namespace (for
   example `site`) in both `web/src/i18n/modules/<namespace>/en-US.ts` and `zh-Hans.ts`. Format prices
   with the locale and currency before passing them (`Intl.NumberFormat`); the block never formats.
3. **Wire actions, not billing.** `action` is a slot: pass a `Button asChild` around a `Link` to the
   sign-up or checkout route. The block has no billing logic; if prices come from an API, load them in
   the page and pass them in.
4. **Billing period toggle (optional).** Put a `ToggleGroup` in `controls` from a small Client
   Component and pass the matching prices per period.
5. **Verify.**
   `cd web && corepack pnpm vitest run src/components/blocks/pricing-table && corepack pnpm type-check && corepack pnpm lint`.
   Then check the layout at 375px, tablet, and desktop width in light and dark mode (`make dev`).
