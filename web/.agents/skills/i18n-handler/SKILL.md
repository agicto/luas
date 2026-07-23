---
name: i18n-handler
description: Change Luas next-intl messages or translation boundaries. Use for locale keys, namespaces, server/client translation APIs, or locale routing.
---

# i18n-handler

## Overview

This skill provides comprehensive instructions for implementing and maintaining internationalization (i18n) within the Luas. It covers client and server component usage, type-safe translation keys, module organization, and scoped translations.

## Key Concepts

### Type System

The i18n system derives key and scope safety from the message tree through:

- **`AllTranslationKeys`**: Union type of all valid dot-notation keys (e.g., `'common.save' | 'auth.login' | ...`)
- **`MessageSchema`**: Literal root schema derived from the `zh-Hans` base locale
- **`AllScopePaths`**: Union of object paths that can be passed to `useT` / `getT`
- **`ScopedTranslationKeys<P>`**: Relative translatable leaf keys below scope `P`
- **`ScopedTranslations<P>`**: Translator constrained to `ScopedTranslationKeys<P>`
- **`TranslationVariables<K>`**: Exact ICU variables derived from the message literal at `K`

Object paths and leaf paths are intentionally distinct. `test.level1` is a valid scope but not a translatable key; `test.level1.title` is a translatable key but not a scope. Keys, scopes, and interpolation variables are derived from `MessageSchema`, so nested message changes update the public types automatically.

## Guidelines

### 1. Unified Translation Pattern

The project uses `next-intl` with a unified pattern that supports both dot notation (recommended) and namespace-based access.

#### Client Components

Use the `useT` hook for translations in client-side components.

```tsx
'use client';

import { useT } from '@/i18n';

function MyComponent() {
  const t = useT();
  return (
    <div>
      {/* Dot notation (recommended) */}
      <button>{t('common.save')}</button>
      
      {/* Namespace-based (backward compatible) */}
      <button>{t.common('save')}</button>
    </div>
  );
}
```

#### Server Components

Use the asynchronous `getT` function for translations in server-side components.

```tsx
// app/page.tsx (Server Component)
import { getT } from '@/i18n/server';

export default async function Page() {
  const t = await getT();
  return <h1>{t('common.loading')}</h1>;
}
```

#### Scoped Translations

For components with many translations from a single namespace, use the scoped pattern:

```tsx
import { useT } from '@/i18n';

function SettingsPage() {
  // Scoped to 'settings' - only relative leaf keys are valid
  const t = useT('settings');
  return (
    <div>
      <h1>{t('title')}</h1>        {/* settings.title */}
      <p>{t('description')}</p>    {/* settings.description */}
    </div>
  );
}
```

### Client Message Boundaries

Server translations can use the complete request message tree. Client Components receive only the namespaces owned by their route:

- `global`: `common`, `errors`
- `auth`: `auth`
- `console`: `auth`, `nav`, `console`
- `i18nTest`: `test`

The canonical registry is `src/i18n/client-message-namespaces.ts`. Route layouts call `selectMessageNamespaces()` and add those messages with `RouteMessagesProvider`. When adding a client-side `useT` namespace, update the nearest route scope and `src/test/i18n-client-messages.test.tsx`. Never pass the full `getMessages()` result back to the root `NextIntlClientProvider`.

Prefer translating in a Server Component and passing the final label to an interactive leaf when that keeps the client independent of a message namespace.

### 2. Module Organization

Translations are organized into functional modules located in `src/i18n/modules/[module]/`.

#### Available Modules

| Module | Description |
|--------|-------------|
| `common` | Common UI text (buttons, labels, messages) |
| `auth` | Authentication-related text |
| `nav` | Navigation labels |
| `site` | Public scaffold site copy |
| `console` | Replaceable authenticated console copy |
| `settings` | Settings page translations |
| `errors` | Error messages |
| `metadata` | Page titles and SEO metadata |
| `test` | Testing translations |

#### Supported Locales

- `zh-Hans` (Simplified Chinese) - **Base locale, defines types**
- `en-US` (English US) - Implements base type

Configuration in `src/i18n/config.ts`.

When adding a configured locale, also add its React DayPicker locale to `src/components/ui/calendar-locale.ts`; the exhaustive `Record<Locale, DayPickerLocale>` makes missing calendar coverage a type error.

### 3. Module Structure

Each module contains three files:

```
src/i18n/modules/[module]/
├── zh-Hans.ts    # Base file (defines types)
├── en-US.ts      # English translations (implements type)
└── index.ts      # Barrel export
```

#### Base File (`zh-Hans.ts`)

```typescript
const messages = {
  title: '标题',
  description: '描述',
  nested: {
    item: '嵌套项目',
  },
} as const;

export default messages;
export type ModuleNameMessages = typeof messages;
```

#### Translation File (`en-US.ts`)

```typescript
import type { ModuleNameMessages } from './zh-Hans';
import type { LocaleMessageShape } from '../../locale-message-shape';

const messages = {
  title: 'Title',
  description: 'Description',
  nested: {
    item: 'Nested Item',
  },
} as const satisfies LocaleMessageShape<ModuleNameMessages>;

export default messages;
```

### 4. Adding New Translations

When adding a new page or feature:

1. Identify the relevant module in `src/i18n/modules/`.
2. Add the translation key and its values to `zh-Hans.ts` first (this defines the type).
3. Add the same key to `en-US.ts`; `LocaleMessageShape` enforces the structure without widening translated literals.
4. Preserve the same ICU variable names in every locale; `modules/index.ts` enforces locale coverage and placeholder parity.

### 5. Adding a New Module

1. Create folder: `modules/[module-name]/`
2. Create `zh-Hans.ts` with type export
3. Create `en-US.ts` implementing the type
4. Create `index.ts` barrel export
5. Register in `modules/index.ts`
6. Add to `AVAILABLE_MODULES` in `src/i18n/module-names.ts`
7. Add dynamic imports to `moduleRegistry` in `loader.ts`

### 6. Using Variables (ICU Format)

```tsx
// In zh-Hans.ts:
// greeting: '你好，{name}！欢迎回来。'

const t = useT();
t('common.greeting', { name: '张三' }); // -> "你好，张三！欢迎回来。"
// or
t.common('greeting', { name: '张三' }); // -> "你好，张三！欢迎回来。"
```

The values object is key-specific. Missing, misspelled, or extra variables fail type checking, including extra properties passed through a previously declared object. Keys without ICU variables do not accept a values object.

## Usage Scenarios

| Task | Action |
|------|--------|
| New Page | Add translations to `src/i18n/modules/[module]/[locale].ts` and update `src/constants/routes.ts`. |
| New Component | Use `useT` from `@/i18n` (Client) or `getT` from `@/i18n/server` (Server) for all user-facing text. |
| Error Messages | Use the `errors` namespace and follow the standardized error handling patterns. |
| Scoped Component | Use `useT('namespace')` to get type-safe scoped translations. |

## File Reference

| File | Purpose |
|------|---------|
| `src/i18n/config.ts` | Locale configuration and settings |
| `src/i18n/locales.ts` | Supported locale constants, names, and Accept-Language mapping |
| `src/i18n/locale-resolution.ts` | Cookie/header/default request locale resolution seam |
| `src/i18n/index.ts` | Client-safe barrel exports for locale config and `useT` |
| `src/i18n/translations.ts` | Client-only `useT` implementation |
| `src/i18n/server.ts` | Server-only `getT` implementation |
| `src/i18n/translation-shared.ts` | Message-tree-derived key/scope types and facade construction |
| `src/i18n/locale-message-shape.ts` | Locale structure and ICU variable-parity type guards |
| `src/i18n/module-names.ts` | Canonical `AVAILABLE_MODULES` declaration |
| `src/i18n/loader.ts` | Dynamic module loading |
| `src/i18n/client-message-namespaces.ts` | Canonical client namespace ownership |
| `src/i18n/message-selection.ts` | Type-safe namespace selection |
| `src/i18n/route-messages-provider.tsx` | Additive route-level client messages |
| `src/i18n/modules/index.ts` | Static type generation from modules |
| `src/i18n/README.md` | Detailed documentation with examples |
| `src/test/i18n-types.test.ts` | Compile-time locale/key/variable contract and runtime composition |

## Environment Variables

The source of truth is `src/config/env.ts`; locale values are constrained by `src/i18n/locales.ts`.

| Variable | Description | Default |
|----------|-------------|---------|
| `NEXT_PUBLIC_DEFAULT_LOCALE` | Default locale | `zh-Hans` |
| `NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED` | Show language switcher | `true` |

Request-time locale detection lives in `src/i18n/locale-resolution.ts`: supported `locale` cookie values win, then `Accept-Language`, then the configured default.

> [!IMPORTANT]
> **Localized Core Copy**: Root metadata, `(site)`, `(auth)`, `(protected)/(console)`, and their shared shell components must resolve user-facing copy through `getT` or `useT`. Prefer server translation when possible. Exact brand names, technical identifiers, and `<code>` content are narrow exceptions. `devtools`, `example`, and the dependency-light `global-error.tsx` fallback are outside the core guard by design. Run `pnpm lint:i18n-copy`; `pnpm lint` includes it.

> [!TIP]
> For comprehensive examples and detailed API documentation, refer to `src/i18n/README.md`.

## Related Skills

Select another skill only when its distinct concern is active.

- [`api-error-handling`](../api-error-handling/): Error messages must be localized.
- [`accessibility-audit`](../accessibility-audit/): `lang` attribute, RTL handling, screen-reader text.
- [`frontend-design`](../frontend-design/): Layouts must handle translation length variation.
- [`ui-styling-guide`](../ui-styling-guide/): Typography metrics differ across scripts.
