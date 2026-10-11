# Midnight Theme

A dark-leaning, high-contrast theme with a vivid violet accent, for developer tools and products
used at night. It restyles both browser shells through design tokens only, so every existing
component, page, and block picks it up without code changes.

```bash
cd api && go run ./cmd/fastbuild plugin:add theme-midnight
```

| Token area | Change |
|---|---|
| Dark mode | Near-black navy surfaces, bright text, clearly visible borders, vivid violet brand |
| Light mode | Midnight-navy primary controls, deep ink text, the same violet brand |
| Radius | `0.5rem` |
| Contrast | Muted text is brighter than the default in dark mode |

It installs two stylesheets and imports them at the theme slots:

- `web/src/themes/presets/midnight.css`, imported at `web.theme` in `web/src/themes/index.css`
- `admin/src/styles/themes/midnight.css`, imported at `admin.theme` in `admin/src/styles/globals.css`

Tokens cannot choose the color scheme: the user's light, dark, or system preference still applies.
To open in dark mode by default, set `defaultTheme="dark"` on the Web `ThemeProvider` in
`web/src/app/layout.tsx` (see `INTEGRATION.md`). The Web and Admin contrast guards (`pnpm lint`)
check every readable pair against WCAG AA 4.5:1. Only one theme can be installed at a time; run
`plugin:remove theme-midnight` to return to the default tokens.
