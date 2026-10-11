# Dashboard Stats Block — Integration Steps

`plugin:add` copied `web/src/components/blocks/dashboard-stats/index.tsx` and its test. Nothing is
wired automatically; complete these steps in order.

1. **Place it.** Render `DashboardStats` in the page or feature component that owns the numbers,
   for example the console home under `web/src/app/(protected)/(console)/console/` or a feature
   under `web/src/features/<feature>/components/`. Data comes from that feature's TanStack Query
   hook or server loader; the block never fetches.
2. **Pass translated copy.** Add every label, trend sentence, and loading label to the owning
   namespace in both `web/src/i18n/modules/<namespace>/en-US.ts` and `zh-Hans.ts`, then pass the
   results of `getT('<namespace>')` (Server Components) or `useT('<namespace>')` (Client Components).
   Format numbers and percentages with the locale (`Intl.NumberFormat` or next-intl formatters)
   before passing them as `value`.
3. **Choose trend meaning.** Set `trend.positive: false` for metrics where an increase is bad (errors,
   latency, churn) so the color matches the meaning.
4. **Verify.**
   `cd web && corepack pnpm vitest run src/components/blocks/dashboard-stats && corepack pnpm type-check && corepack pnpm lint`.
   Then open the page at 375px and desktop width in light and dark mode (`make dev`).
