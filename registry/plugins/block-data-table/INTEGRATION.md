# Data Table Block — Integration Steps

`plugin:add` copied `web/src/components/blocks/data-table/index.tsx` and its test. Nothing is wired
automatically; complete these steps in order.

1. **Place it in a Client Component.** Columns are render functions, which cannot cross the
   Server-to-Client boundary. Create (or reuse) a `'use client'` component in the owning feature,
   for example `web/src/features/<feature>/components/<name>-table.tsx`, define the typed columns
   there, and pass rows from the feature's TanStack Query hook.
2. **Pass translated copy.** Add the caption, column headers, pagination labels, page status (an ICU
   message with `{page}` and `{pageCount}`), and empty-state text to the owning namespace in both
   `web/src/i18n/modules/<namespace>/en-US.ts` and `zh-Hans.ts`; read them with `useT('<namespace>')`.
   Format dates and numbers with the locale inside `cell`.
3. **Pick the data mode.** The block sorts and paginates the rows it receives, which suits lists of up
   to a few thousand rows. For server-side pagination, pass one page of rows with `pageSize={0}`, render
   your own pagination from the API's pagination fields (see `contracts/README.md`), and use
   `onSortChange` to request a new order.
4. **Empty and error states.** Pass a short message or the `block-empty-state` block with `bare` as
   `emptyState`. Render errors outside the table with the feature's existing error handling.
5. **Verify.**
   `cd web && corepack pnpm vitest run src/components/blocks/data-table && corepack pnpm type-check && corepack pnpm lint`.
   Then check keyboard sorting (Tab to a header, Enter) and horizontal scrolling at 375px in light and
   dark mode (`make dev`).
