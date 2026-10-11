# Enterprise Theme

A calm blue theme for business and internal tools. It restyles both browser shells through design
tokens only, so every existing component, page, and block picks it up without code changes.

```bash
cd api && go run ./cmd/fastbuild plugin:add theme-enterprise
```

| Token area | Change |
|---|---|
| Color | Calm blue primary, brand, and focus ring; cool neutral surfaces |
| Borders | Opaque, slightly darker borders and inputs for crisp edges |
| Radius | `0.375rem` (default `0.625rem` on Web, `0.5rem` on Admin) |
| Density | Tailwind spacing unit `0.24rem` (default `0.25rem`), about 4% denser |
| Admin material | Less translucent cards, panels, and sidebar |

It installs two stylesheets and imports them at the theme slots:

- `web/src/themes/presets/enterprise.css`, imported at `web.theme` in `web/src/themes/index.css`
- `admin/src/styles/themes/enterprise.css`, imported at `admin.theme` in `admin/src/styles/globals.css`

Light and dark mode are both covered. The Web and Admin contrast guards (`pnpm lint`) check every
readable pair against WCAG AA 4.5:1. Only one theme can be installed at a time; run
`plugin:remove theme-enterprise` to return to the default tokens.
