# Empty State Block — Integration Steps

`plugin:add` copied `web/src/components/blocks/empty-state/index.tsx` and its test. Nothing is wired
automatically; complete these steps in order.

1. **Place it where the state happens.** Render it in the feature component that knows the list is
   empty, the search matched nothing, the query failed, or access was denied, for example in
   `web/src/features/<feature>/components/`. Inside a card or a table cell use `bare`; as a page
   body use the default frame.
2. **Map errors to copy, never to backend text.** For `variant="error"` and `variant="no-access"`,
   choose the title and description from the feature's i18n error mapping based on the error code
   (`ApiErrorCode` / `ClientErrorCode`, see `web/src/http/`). Show `request_id` in `detail` when present.
   Do not render upstream error messages.
3. **Pass translated copy.** Add the titles, descriptions, and action labels to the owning namespace in
   both `web/src/i18n/modules/<namespace>/en-US.ts` and `zh-Hans.ts`; pass them from
   `getT('<namespace>')` or `useT('<namespace>')`.
4. **Heading level.** Pick `headingLevel` so the outline has no gaps (for example `h3` under a card
   title that is an `h2`).
5. **Verify.**
   `cd web && corepack pnpm vitest run src/components/blocks/empty-state && corepack pnpm type-check && corepack pnpm lint`.
   Then view each state at 375px and desktop width in light and dark mode (`make dev`).
