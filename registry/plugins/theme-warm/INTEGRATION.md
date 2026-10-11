# Warm Theme — Integration Steps

`plugin:add` already copied `web/src/themes/presets/warm.css` and `admin/src/styles/themes/warm.css`
and imported them at the `web.theme` and `admin.theme` slots. A theme needs no code changes;
complete these checks in order.

1. **Confirm the imports.** `web/src/themes/index.css` imports `./presets/warm.css` after
   `./dark.css`, and `admin/src/styles/globals.css` imports `./themes/warm.css` after
   `./tokens.css`. `cd api && go run ./cmd/fastbuild plugin:doctor` reports no problems.
2. **Check contrast and the build.**
   `cd web && corepack pnpm lint && corepack pnpm type-check`, then
   `cd admin && corepack pnpm lint && corepack pnpm build`. Both contrast guards print the pairs they
   checked; they fail when a pair drops below 4.5:1.
3. **Review reading surfaces.** Open long-form pages (articles, settings descriptions, empty states)
   in light and dark mode at 375px and desktop width and confirm text, links, and badges read
   comfortably on the warm surfaces (`make dev`).
4. **Adapt brand colors only through tokens.** To move the accent toward the brand, edit the
   `oklch()` values in the two theme files (keep `:root` and `.dark` in sync) and rerun step 2. Do not
   hard-code colors in components.

Ask the builder before changing typography: the theme keeps the scaffold's font stacks, which are
compiled into Tailwind utilities rather than exposed as tokens.
