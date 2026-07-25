# Internationalization

The static SPA uses `i18next` and `react-i18next`. Its locale contract matches
the Next.js Web shell and shared settings API while remaining fully
browser-hosted.

## Supported Locales

| Locale    | Language                | Role                                 |
| --------- | ----------------------- | ------------------------------------ |
| `en-US`   | English (United States) | Canonical source and fallback locale |
| `zh-Hans` | Simplified Chinese      | Supported translation locale         |

Locale identifiers use BCP 47 tags. Do not introduce SPA-only aliases such as
`en` or `zh-CN` into persisted state or API values. Browser aliases are
normalized at the locale-resolution boundary.

## Resolution And Persistence

The initial locale is selected in this order:

1. The `luas-spa-locale` browser preference.
2. `navigator.languages` in browser preference order.
3. `VITE_DEFAULT_LOCALE`.

English variants normalize to `en-US`; Chinese variants normalize to
`zh-Hans`. Changing language updates the document `lang` attribute and stores
the locale in `localStorage`.

Locale is display preference state. Never store credentials, authorization
state, server-owned profile data, or sensitive attributes beside it.

## Source Layout

```text
src/i18n/
├── resources.ts     Locale messages and `SupportedLocale`
├── locale.ts        Pure normalization and preference resolution
├── locale.test.ts   Locale contract tests
└── index.ts         i18next initialization and browser lifecycle
```

The current catalog is intentionally compact for the lightweight shell. Split
it into feature-owned namespaces or lazy resources when catalog size becomes a
measured bundle concern; preserve the same locale IDs and key ownership.

The `en-US` resource defines the source shape. TypeScript rejects missing or
misplaced `zh-Hans` keys, and a compile-time parity check requires translated
i18next interpolation variables such as `{{time}}` to retain their source
names.

## Adding Messages

1. Add semantic keys to the `en-US` resource.
2. Add the same tree to `zh-Hans`.
3. Use complete messages instead of concatenated translated fragments.
4. Use i18next interpolation for dynamic values.
5. Keep backend messages out of UI copy; map stable `error_code` values to
   localized messages.

Use `useTranslation()` in components and pass translated final labels into
feature-neutral primitives where practical.

## Adding A Locale

1. Add a BCP 47 resource key and native language label.
2. Cover the complete formal UI message tree.
3. Extend `SupportedLocale` through the resource key.
4. Add normalization rules only for valid aliases.
5. Update the typed `VITE_DEFAULT_LOCALE` schema and `.env.example`.
6. Add locale resolution, formatting, and visible-copy tests.

Do not publish a partially translated formal console. A downstream product may
define an explicit fallback policy when partial rollout is a product decision.

## Formatting And Accessibility

- Pass `i18n.language` to `Intl.DateTimeFormat` and other `Intl` formatters.
- Keep machine values independent from localized display strings.
- Set the document `lang` attribute whenever the locale changes.
- Test controls with longer translations and narrow viewports.
- Use translated accessible names for icon-only controls.
- Treat RTL as a separately tested capability before adding an RTL locale.

## Configuration

```env
VITE_DEFAULT_LOCALE=en-US
```

`VITE_*` values are public build-time configuration. They must not contain
secrets.

## Verification

```bash
corepack pnpm vitest run src/i18n/locale.test.ts
corepack pnpm type-check
corepack pnpm lint
corepack pnpm build
```

The production build remains static and can be deployed to OSS/S3-compatible
object storage and a CDN.
