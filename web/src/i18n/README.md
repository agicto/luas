# i18n Module

Internationalization (i18n) module using `next-intl` with TypeScript message files.

## Key Concepts

### Type System

The i18n system derives key and scope safety from the message tree through:

- **`AllTranslationKeys`**: Union type of all valid dot-notation keys (e.g., `'common.save' | 'auth.login' | ...`)
- **`Messages`**: Root type containing all module namespaces and their translations
- **`AllScopePaths`**: Union of valid object paths accepted by scoped translators
- **`ScopedTranslationKeys<P>`**: Relative translatable leaf keys below scope `P`
- **`ScopedTranslations<P>`**: Translator constrained to `ScopedTranslationKeys<P>`
- **`UnifiedTranslations`**: Combined type that supports both dot notation and namespace accessors

Scope paths and translation keys are separate unions derived from `Messages`. Object nodes such as `test.level1` are valid scopes but cannot be translated directly; leaf paths such as `test.level1.title` are valid translation keys but cannot be used as scopes.

## Structure

```
i18n/
├── config.ts           # Locale configuration + ENV variables
├── request.ts          # next-intl server configuration
├── index.ts            # Client-safe barrel exports
├── translations.ts     # Client-only useT implementation
├── server.ts           # Server-only getT implementation
├── translation-shared.ts # Message-tree-derived types and pure helpers
├── module-names.ts     # Canonical translation module names
├── loader.ts           # Dynamic module loading
├── client-message-namespaces.ts # Client namespace ownership by route
├── message-selection.ts # Type-safe top-level namespace selection
├── route-messages-provider.tsx # Additive nested client provider
└── modules/            # Translation modules
    ├── common/         # Common translations (buttons, labels)
    ├── auth/           # Authentication translations
    ├── nav/            # Navigation translations
    ├── settings/       # Settings translations
    ├── errors/         # Error messages
    ├── metadata/       # Page metadata
    ├── dashboard/      # Dashboard translations
    └── test/           # Testing translations
```

Each module contains:
- `zh-Hans.ts` - Simplified Chinese (base, defines types)
- `en-US.ts` - English US (implements base type)
- `index.ts` - Barrel export

## Adding a New Translation Key

1. Add key to `zh-Hans.ts` (base file):
```typescript
// modules/common/zh-Hans.ts
const messages = {
  loading: '加载中...',
  newKey: '新的键', // Add here
};
```

2. Add to other locale files (TypeScript will enforce this):
```typescript
// modules/common/en-US.ts
const messages: CommonMessages = {
  loading: 'Loading...',
  newKey: 'New Key', // Must add - enforced by type
};
```

## Adding a New Module

1. Create folder: `modules/[module-name]/`
2. Create `zh-Hans.ts`:
```typescript
const messages = {
  key1: '键1',
  key2: '键2',
};
export default messages;
export type ModuleNameMessages = typeof messages;
```

3. Create `en-US.ts`:
```typescript
import type { ModuleNameMessages } from './zh-Hans';
const messages: ModuleNameMessages = {
  key1: 'Key 1',
  key2: 'Key 2',
};
export default messages;
```

4. Create `index.ts`:
```typescript
export { default as zhHans } from './zh-Hans';
export { default as enUS } from './en-US';
```

5. Register in `modules/index.ts`:
```typescript
import * as moduleName from './module-name';

const modules = {
  // ...existing
  moduleName,
};

// Add to getMessages return
return {
  // ...existing
  moduleName: modules.moduleName[exportKey],
};
```

## Adding a New Locale

1. Add locale to `config.ts`:
```typescript
export const locales = ['zh-Hans', 'en-US', 'ja-JP'] as const;

export const localeNames: Record<Locale, string> = {
  // ...existing
  'ja-JP': '日本語',
};

export const localeMapping: Record<string, Locale> = {
  // ...existing
  'ja': 'ja-JP',
  'ja-JP': 'ja-JP',
};
```

2. Update `modules/index.ts`:
```typescript
const localeToExport: Record<Locale, 'zhHans' | 'enUS' | 'jaJP'> = {
  // ...existing
  'ja-JP': 'jaJP',
};
```

3. Add `ja-JP.ts` to each module implementing the base type.

## Environment Variables

Locale behavior is configured through typed env values in `src/config/env.ts`, constrained by `src/i18n/locales.ts`, and consumed by `src/i18n/config.ts`.

| Variable | Default | Description |
|---|---|---|
| `NEXT_PUBLIC_DEFAULT_LOCALE` | `zh-Hans` | Default locale. Must be one of `locales`. |
| `NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED` | `true` | Shows or hides the language switcher. |

Request-time locale detection lives in `src/i18n/locale-resolution.ts`: supported `locale` cookie values win, then `Accept-Language`, then `NEXT_PUBLIC_DEFAULT_LOCALE`.

## Usage

### Unified Translation Function ✨

The unified `useT` hook provides clean, type-safe access to all translation modules.

#### Client Components

```tsx
'use client';

import { useT } from '@/i18n';

export function LoginForm() {
  const t = useT();

  return (
    <form>
      <label>{t.auth('email')}</label>
      <input type="email" />
      <button type="submit">{t.common('save')}</button>
      <p>{t.errors('networkError')}</p>
    </form>
  );
}
```

#### Server Components

```tsx
// app/page.tsx (Server Component)
import { getT } from '@/i18n/server';

export default async function Page() {
  const t = await getT();

  return (
    <nav>
      <a href="/">{t.nav('home')}</a>
      <a href="/dashboard">{t.nav('dashboard')}</a>
      <span>{t.common('loading')}</span>
    </nav>
  );
}
```

#### Scoped Translations

For components with many translations from a single namespace, use the scoped pattern for cleaner code:

```tsx
'use client';

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

Server-side scoped translations:

```tsx
import { getT } from '@/i18n/server';

export default async function ErrorPage() {
  const t = await getT('errors');
  return <p>{t('networkError')}</p>;  {/* errors.networkError */}
}
```

Keep server imports explicit. `getT` is intentionally not re-exported from the client-safe
`@/i18n` barrel, so a Client Component cannot accidentally pull in `next-intl/server`.

### Client Message Delivery

`src/i18n/request.ts` loads the complete message tree for Server Components. The browser receives a smaller tree:

| Scope | Added namespaces | Owner |
|---|---|---|
| `global` | `common`, `errors` | Root layout and route error UI |
| `auth` | `auth` | Login/register route group |
| `console` | `auth`, `nav` | Console route group |
| `i18nTest` | `test` | i18n devtool route |

`CLIENT_MESSAGE_NAMESPACES` is the source of truth. `selectMessageNamespaces()` picks full top-level namespaces on the server, while `RouteMessagesProvider` merges route-owned namespaces with the inherited global messages on the client.

When a Client Component starts using another namespace:

1. Add the namespace to the nearest route scope in `client-message-namespaces.ts`.
2. Ensure the owning route layout selects it and renders `RouteMessagesProvider`.
3. Update `src/test/i18n-client-messages.test.tsx`.
4. Run a production build and confirm unrelated routes do not serialize the new namespace.

Do not pass the complete `getMessages()` result to the root `NextIntlClientProvider`; that sends every feature and devtool message to every route.

### With Variables (ICU Format)

```tsx
// If you add a key with variables in zh-Hans.ts:
// greeting: '你好，{name}！欢迎回来。'

const t = useT();
t('common.greeting', { name: '张三' }); // -> "你好，张三！欢迎回来。"
// or
t.common('greeting', { name: '张三' }); // -> "你好，张三！欢迎回来。"
```

### Switching Locale Programmatically

```tsx
'use client';

import { useLocale } from '@/hooks/use-locale';

export function LanguageButton() {
  const { locale, setLocale } = useLocale();

  return (
    <button onClick={() => setLocale(locale === 'zh-Hans' ? 'en-US' : 'zh-Hans')}>
      Switch Language
    </button>
  );
}
```

## Environment Variables

Locale behavior is configured through typed env values in `src/config/env.ts`, constrained by `src/i18n/locales.ts`, and consumed by `src/i18n/config.ts`.

| Variable | Default | Description |
|---|---|---|
| `NEXT_PUBLIC_DEFAULT_LOCALE` | `zh-Hans` | Default locale. Must be one of `locales`. |
| `NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED` | `true` | Shows or hides the language switcher. |

Request-time locale detection lives in `src/i18n/locale-resolution.ts`: supported `locale` cookie values win, then `Accept-Language`, then `NEXT_PUBLIC_DEFAULT_LOCALE`.

## Type Safety

- Base locale (`zh-Hans.ts`) defines the message structure
- Other locales must implement the same structure (enforced by TypeScript)
- `AllTranslationKeys` contains only leaf messages, never object nodes
- `AllScopePaths` contains only object paths, never leaf messages
- `ScopedTranslations<P>` accepts only relative leaf keys below `P`
- Missing or misplaced keys cause compile-time errors and are guarded by `src/test/i18n-types.test.ts`
