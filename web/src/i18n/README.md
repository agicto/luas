# Internationalization

The Next.js application uses `next-intl` with TypeScript message modules. The
design keeps locale selection deterministic, translation keys type-safe, ICU
arguments aligned, and client message payloads bounded by route ownership.

## Supported Locales

| Locale    | Language                | Role                                       |
| --------- | ----------------------- | ------------------------------------------ |
| `en-US`   | English (United States) | Canonical source locale and message schema |
| `zh-Hans` | Simplified Chinese      | Fully supported translation locale         |

Locale identifiers use BCP 47 language tags. Use `zh-Hans` for the script-level
Simplified Chinese translation instead of tying the content to one region.

The source locale and default locale are separate concerns. `en-US` defines the
message structure in source control; `NEXT_PUBLIC_DEFAULT_LOCALE` controls the
request fallback and defaults to `zh-Hans` in the scaffold.

## Runtime Resolution

The server resolves a request locale in this order:

1. A valid `locale` cookie.
2. The highest-quality supported language in `Accept-Language`.
3. `NEXT_PUBLIC_DEFAULT_LOCALE`.

Unsupported and `q=0` language ranges are ignored. Regional English variants
map to `en-US`; supported Chinese variants map to `zh-Hans`. The resolved locale
drives the document `lang`, message loading, and locale-aware formatters.

Configuration is validated in `src/config/env.ts`:

| Variable                              | Default   | Purpose                      |
| ------------------------------------- | --------- | ---------------------------- |
| `NEXT_PUBLIC_DEFAULT_LOCALE`          | `zh-Hans` | Request fallback locale      |
| `NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED` | `true`    | Language-switcher visibility |

## Architecture

```text
src/i18n/
├── locales.ts                    Supported locales, names, and aliases
├── locale-resolution.ts          Cookie and Accept-Language resolution
├── config.ts                     Validated public configuration
├── request.ts                    next-intl request integration
├── loader.ts                     Locale and namespace module registry
├── module-names.ts               Canonical namespace catalog
├── modules/<namespace>/
│   ├── en-US.ts                  Canonical source messages and schema type
│   ├── zh-Hans.ts                Structurally checked translation
│   └── index.ts                  Stable locale exports
├── translation-shared.ts         Typed key, scope, and ICU argument APIs
├── locale-message-shape.ts       Shape and ICU parity contracts
├── client-message-namespaces.ts  Route-owned browser message budgets
├── message-selection.ts          Namespace selection
└── route-messages-provider.tsx   Additive client message provider
```

`MessageSchema` is derived from the `en-US` modules. Other locales use
`LocaleMessageShape<...>` so missing, extra, or incorrectly nested keys fail
type checking. `LocaleMessageVariableParity` also requires every locale to use
the same ICU argument names.

## Translation APIs

Use server translation unless the component must translate during browser
interaction.

```tsx
import { getT } from '@/i18n/server';

export default async function Page() {
  const t = await getT('settings');
  return <h1>{t('title')}</h1>;
}
```

Client Components use the client-safe `useT` export:

```tsx
'use client';

import { useT } from '@/i18n';

export function SaveButton() {
  const t = useT('common');
  return <button type="submit">{t('save')}</button>;
}
```

Unscoped dot notation is useful when a component owns a few messages from
different namespaces:

```tsx
const t = useT();
t('common.save');
t('errors.networkError');
```

ICU variables are key-specific and checked by TypeScript:

```tsx
t('common.greeting', { name: 'Alex' });
```

Do not import `@/i18n/server` from a Client Component. Do not render backend
`message` values as user copy; map stable `error_code` values to the `errors`
namespace.

## Client Message Ownership

Server Components may load the complete request message tree. Browser routes
receive only the namespaces required by their interactive leaves.

`CLIENT_MESSAGE_NAMESPACES` is the authority for this payload boundary.
`selectMessageNamespaces()` selects whole top-level namespaces on the server,
and `RouteMessagesProvider` adds them to inherited global messages.

When a Client Component needs a new namespace:

1. Add it to the nearest route scope in `client-message-namespaces.ts`.
2. Select it in the owning route layout.
3. Render `RouteMessagesProvider` around the interactive subtree.
4. Update `src/test/i18n-client-messages.test.tsx`.
5. Build and confirm unrelated routes do not serialize that namespace.

Never pass the full `getMessages()` result to the root client provider.

## Adding A Message

1. Choose the feature-owned namespace; avoid a generic dumping-ground module.
2. Add the key and English source copy to `en-US.ts`.
3. Add the same structure and ICU arguments to `zh-Hans.ts`.
4. Use semantic keys such as `invite.expired`, not copy-derived keys such as
   `invitationHasExpired`.
5. Use the key through `getT` or `useT`; do not add user-facing literals to
   formal scaffold surfaces.

Prefer complete sentences for messages. Keep punctuation in translations,
avoid concatenating translated fragments, and use ICU variables for dynamic
values.

## Adding A Namespace

1. Create `modules/<namespace>/en-US.ts`, `zh-Hans.ts`, and `index.ts`.
2. Export the source message type from `en-US.ts`.
3. Make `zh-Hans.ts` satisfy `LocaleMessageShape<SourceMessages>`.
4. Add the namespace to `AVAILABLE_MODULES`.
5. Register both dynamic imports in `loader.ts`.
6. Register the source message and schema in `modules/index.ts`.
7. Add client ownership only when an interactive route needs it.

Namespace names should follow domain ownership (`organization`, `webhook`) or
stable shell ownership (`common`, `nav`, `metadata`).

## Adding A Locale

1. Add its BCP 47 tag and native display name to `locales.ts`.
2. Add a locale file to every registered namespace.
3. Register every locale/namespace loader in `loader.ts`.
4. Add its schema to `LocaleMessageSchemas` in `modules/index.ts`.
5. Add the matching React DayPicker locale in
   `src/components/ui/calendar-locale.ts`.
6. Add resolution, ICU parity, date, number, and visible-copy tests.

Do not add a locale with partial formal-surface coverage. Product-specific
fallback policies belong in the downstream application and must be explicit.

## Formatting And Layout

- Use `Intl.DateTimeFormat`, `Intl.NumberFormat`, and `Intl.RelativeTimeFormat`
  with the resolved locale.
- Store timestamps and machine values independently from localized display
  strings.
- Allow controls and tables to accommodate longer translations.
- Use logical CSS properties for direction-sensitive layout.
- Set accessible names through translated messages.
- Introduce RTL support as a tested application capability, not a CSS toggle.

## Verification

```bash
corepack pnpm vitest run src/test/locale-resolution.test.ts
corepack pnpm vitest run src/test/i18n-types.test.ts
corepack pnpm vitest run src/test/i18n-client-messages.test.tsx
corepack pnpm lint:i18n-copy
corepack pnpm type-check
corepack pnpm build
```

The standard `pnpm lint` command includes the formal-copy guard. Repository
release validation runs through `make check` at the root.
