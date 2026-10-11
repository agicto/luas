# Midnight Theme — Integration Steps

`plugin:add` already copied `web/src/themes/presets/midnight.css` and
`admin/src/styles/themes/midnight.css` and imported them at the `web.theme` and `admin.theme`
slots. A theme needs no code changes; complete these checks in order.

1. **Confirm the imports.** `web/src/themes/index.css` imports `./presets/midnight.css` after
   `./dark.css`, and `admin/src/styles/globals.css` imports `./themes/midnight.css` after
   `./tokens.css`. `cd api && go run ./cmd/fastbuild plugin:doctor` reports no problems.
2. **Check contrast and the build.**
   `cd web && corepack pnpm lint && corepack pnpm type-check`, then
   `cd admin && corepack pnpm lint && corepack pnpm build`. Both contrast guards print the pairs they
   checked; they fail when a pair drops below 4.5:1.
3. **Optional: open in dark mode by default.** Only if the builder wants it, change
   `defaultTheme="system"` to `defaultTheme="dark"` on the `ThemeProvider` in
   `web/src/app/layout.tsx`; users can still switch. Verify with
   `cd web && corepack pnpm vitest run src/test/theme-toggle.test.tsx`. The Admin Console keeps
   following the user's saved preference.
4. **Review accents.** Open pages with badges, links, charts, and focus rings in both modes at 375px
   and desktop width (`make dev`) and confirm the violet accent stays readable and focus is visible.
5. **Adapt brand colors only through tokens.** Edit the `oklch()` values in the two theme files
   (keep `:root` and `.dark` in sync) and rerun step 2. Do not hard-code colors in components.
