# Warm Theme

A content-first theme for documentation, knowledge bases, blogs, and communities. It restyles both
browser shells through design tokens only, so every existing component, page, and block picks it
up without code changes.

```bash
cd api && go run ./cmd/fastbuild plugin:add theme-warm
```

| Token area | Change |
|---|---|
| Color | Warm paper neutrals, ink-like text, terracotta brand and focus ring |
| Borders | Soft, warm borders that recede behind content |
| Radius | `0.875rem` on Web and `0.75rem` on Admin (defaults `0.625rem` and `0.5rem`) |
| Status | Error and success text darkened slightly so they stay readable on warm surfaces |

It installs two stylesheets and imports them at the theme slots:

- `web/src/themes/presets/warm.css`, imported at `web.theme` in `web/src/themes/index.css`
- `admin/src/styles/themes/warm.css`, imported at `admin.theme` in `admin/src/styles/globals.css`

Light and dark mode are both covered. The Web and Admin contrast guards (`pnpm lint`) check every
readable pair against WCAG AA 4.5:1. Only one theme can be installed at a time; run
`plugin:remove theme-warm` to return to the default tokens.
