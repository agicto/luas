# Onboarding Steps Block — Integration Steps

`plugin:add` copied `web/src/components/blocks/onboarding-steps/index.tsx` and its test. Nothing is
wired automatically; complete these steps in order.

1. **Place it.** Usually on the console home (`web/src/app/(protected)/(console)/console/page.tsx`)
   or a feature landing page, until every step is complete. Hide it, or replace it with a short
   success message, once all steps are complete.
2. **Derive status from real state.** Compute each step's status from data the feature already has
   (for example the profile, organization members, or API keys queries). Exactly one step should be
   `current`. The block stores nothing; do not persist checklist state in the browser as the source of
   truth.
3. **Pass translated copy.** Add the title, step titles and descriptions, action labels, the progress
   message (an ICU message such as `{done} of {total} steps complete`), and the three status labels to
   the owning namespace in both `web/src/i18n/modules/<namespace>/en-US.ts` and `zh-Hans.ts`.
4. **Verify.**
   `cd web && corepack pnpm vitest run src/components/blocks/onboarding-steps && corepack pnpm type-check && corepack pnpm lint`.
   Then check the list with a screen reader or the accessibility tree (current step and status
   announced) at 375px and desktop width in light and dark mode (`make dev`).
