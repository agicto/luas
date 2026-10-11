# Enterprise Theme — Integration Steps

`plugin:add` already copied `web/src/themes/presets/enterprise.css` and
`admin/src/styles/themes/enterprise.css` and imported them at the `web.theme` and `admin.theme`
slots. A theme needs no code changes; complete these checks in order.

1. **Confirm the imports.** `web/src/themes/index.css` imports `./presets/enterprise.css` after
   `./dark.css`, and `admin/src/styles/globals.css` imports `./themes/enterprise.css` after
   `./tokens.css`. `cd api && go run ./cmd/fastbuild plugin:doctor` reports no problems.
2. **Check contrast and the build.**
   `cd web && corepack pnpm lint && corepack pnpm type-check`, then
   `cd admin && corepack pnpm lint && corepack pnpm build`. Both contrast guards print the pairs they
   checked; they fail when a pair drops below 4.5:1.
3. **Review dense screens.** The spacing unit is 4% smaller. Open the busiest console pages and
   tables at 375px and desktop width in light and dark mode and confirm nothing truncates or wraps
   unexpectedly (`make dev`, then the Web console and the Admin Console).
4. **Adapt brand colors only through tokens.** To shift the hue for a brand, edit the `oklch()`
   values in the two theme files (keep `:root` and `.dark` in sync) and rerun step 2. Do not hard-code
   colors in components.

Ask the builder before also changing `defaultTheme` or component variants: the theme deliberately
changes tokens only.
