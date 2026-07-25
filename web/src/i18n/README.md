# i18n Module

Internationalization (i18n) module using `next-intl` with TypeScript message files.

## Key Concepts

### Type System

The i18n system derives key and scope safety from the message tree through:

- **`MessageSchema`**: Canonical literal schema derived from the `zh-Hans` base locale
- **`AllTranslationKeys`**: Union type of all valid dot-notation keys (e.g., `'common.save' | 'auth.login' | ...`)
- **`Messages`**: Root type containing all module namespaces and their translations
- **`AllScopePaths`**: Union of valid object paths accepted by scoped translators
- **`ScopedTranslationKeys<P>`**: Relative translatable leaf keys below scope `P`
- **`ScopedTranslations<P>`**: Translator constrained to `ScopedTranslationKeys<P>`
- **`TranslationVariables<K>`**: Exact ICU variable object derived from the base-locale message literal at `K`
- **`UnifiedTranslations`**: Combined type that supports both dot notation and namespace accessors

Scope paths, translation keys, and interpolation variables are derived from `MessageSchema`. Object nodes such as `test.level1` are valid scopes but cannot be translated directly; leaf paths such as `test.level1.title` are valid translation keys but cannot be used as scopes.

## Structure

```
i18n/
├── config.ts           # Locale configuration + ENV variables
├── request.ts          # next-intl server configuration
├── index.ts            # Client-safe barrel exports
├── translations.ts     # Client-only useT implementation
├── server.ts           # Server-only getT implementation
├── translation-shared.ts # Message-tree-derived types and pure helpers
├── locale-message-shape.ts # Locale shape and ICU variable parity types
├── module-names.ts     # Canonical translation module names
├── loader.ts           # Dynamic module loading
├── client-message-namespaces.ts # Client namespace ownership by route
├── message-selection.ts # Type-safe top-level namespace selection
├── route-messages-provider.tsx # Additive nested client provider
└── modules/            # Translation modules
    ├── common/         # Common translations (buttons, labels)
    ├── auth/           # Authentication translations
    ├── nav/            # Navigation translations
    ├── site/           # Public scaffold site copy
    ├── console/        # Replaceable authenticated console copy
    ├── settings/       # Settings translations
    ├── errors/         # Error messages
    ├── metadata/       # Page metadata
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
  loading: 'Loading...',
  newKey: 'New key', // Add here
} as const;
```

2. Add to other locale files (TypeScript will enforce this):
```typescript
// modules/common/en-US.ts
import type { LocaleMessageShape } from '../../locale-message-shape';

const messages = {
  loading: 'Loading...',
  newKey: 'New Key', // Must add - enforced by type
} as const satisfies LocaleMessageShape<CommonMessages>;
```

## Adding a New Module

1. Create folder: `modules/[module-name]/`
2. Create `zh-Hans.ts`:
```typescript
const messages = {
  key1: 'Key 1',
  key2: 'Key 2',
} as const;
export default messages;
export type ModuleNameMessages = typeof messages;
```

3. Create `en-US.ts`:
```typescript
import type { ModuleNameMessages } from './zh-Hans';
import type { LocaleMessageShape } from '../../locale-message-shape';

const messages = {
  key1: 'Key 1',
  key2: 'Key 2',
} as const satisfies LocaleMessageShape<ModuleNameMessages>;
export default messages;
```

4. Create `index.ts`:
```typescript
export { default as zhHans } from './zh-Hans';
export { default as enUS } from './en-US';
```

5. Register the type source in `modules/index.ts`:
```typescript
import moduleName from './module-name/en-US';

export const messages = {
  // existing modules
  moduleName,
} as const;
```

6. Add the name to `AVAILABLE_MODULES` in `module-names.ts`.
7. Add both locale imports to `moduleRegistry` in `loader.ts`.

## Adding a New Locale

1. Add locale to `config.ts`:
```typescript
export const locales = ['zh-Hans', 'en-US', 'ja-JP'] as const;

export const localeNames: Record<Locale, string> = {
  // ...existing
  'ja-JP': 'Japanese',
};

export const localeMapping: Record<string, Locale> = {
  // ...existing
  'ja': 'ja-JP',
  'ja-JP': 'ja-JP',
};
```

2. Add `ja-JP.ts` to every module using `as const satisfies LocaleMessageShape<BaseMessages>` and export its `typeof messages` type.
3. Add a type-only `ja-JP` schema entry to `LocaleMessageSchemas` in `modules/index.ts`. The coverage guard fails until every configured locale is registered, and the parity guard fails if any ICU variable name differs from the base locale.
4. Add each `ja-JP` dynamic import to `moduleRegistry` in `loader.ts`. Its `Record<Locale, ModuleLoader>` contract fails until every module covers the new locale.

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
      <a href="/console">{t.nav('console')}</a>
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

function SettingsSystemPanel() {
  // Scoped to 'settings.system' - only relative leaf keys are valid
  const t = useT('settings.system');
  return (
    <div>
      <h1>{t('title')}</h1>        {/* settings.system.title */}
      <p>{t('description')}</p>    {/* settings.system.description */}
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
| `console` | `auth`, `nav`, `console` | Console route group |
| `i18nTest` | `test` | i18n devtool route |

`CLIENT_MESSAGE_NAMESPACES` is the source of truth. `selectMessageNamespaces()` picks full top-level namespaces on the server, while `RouteMessagesProvider` merges route-owned namespaces with the inherited global messages on the client.

When a Client Component starts using another namespace:

1. Add the namespace to the nearest route scope in `client-message-namespaces.ts`.
2. Ensure the owning route layout selects it and renders `RouteMessagesProvider`.
3. Update `src/test/i18n-client-messages.test.tsx`.
4. Run a production build and confirm unrelated routes do not serialize the new namespace.

Do not pass the complete `getMessages()` result to the root `NextIntlClientProvider`; that sends every feature and devtool message to every route.

### Core Copy Boundary

The copy guard follows Luas surface ownership rather than scanning every demonstration file:

- Root metadata, `(site)`, `(auth)`, `(protected)/(console)`, and shared shell components must use `getT` or `useT` for user-facing copy.
- Prefer server translation and pass final labels to interactive leaves when that avoids another client namespace.
- Exact brand names, technical identifiers, and text inside `<code>` are allowed literal values.
- `devtools` and `example` are disposable surfaces and are excluded. They must not leak copy into formal scaffold surfaces.
- `global-error.tsx` is excluded because it must remain dependency-light when the normal root runtime fails.
- UI primitive default labels are controlled through component APIs; formal callers should pass translated accessible labels.

Run `pnpm lint:i18n-copy`. The standard `pnpm lint` command includes this guard.

### With Variables (ICU Format)

```tsx
// If you add a key with variables in zh-Hans.ts:
// greeting: 'Hello, {name}! Welcome back.'

const t = useT();
t('common.greeting', { name: 'Alex' }); // -> "Hello, Alex! Welcome back."
// or
t.common('greeting', { name: 'Alex' }); // -> "Hello, Alex! Welcome back."

// Compile-time errors: missing, misspelled, or extra variables.
t('common.greeting');
t('common.greeting', { user: 'Alex' });
t('common.greeting', { name: 'Alex', tenant: 'Luas' });
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

- Base locale (`zh-Hans.ts`) defines the literal message schema with `as const`
- Other locales preserve literals with `as const satisfies LocaleMessageShape<BaseMessages>`
- Configured locale coverage and ICU variable-name parity are enforced in `modules/index.ts`
- `AllTranslationKeys` contains only leaf messages, never object nodes
- `AllScopePaths` contains only object paths, never leaf messages
- `ScopedTranslations<P>` accepts only relative leaf keys below `P`
- Variable-bearing messages require exactly their declared ICU arguments; messages without variables reject a values object
- Missing or misplaced keys, locale drift, and interpolation mistakes are guarded by `src/test/i18n-types.test.ts`
