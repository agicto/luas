# i18n Module

Internationalization (i18n) module using `next-intl` with TypeScript message files.

## Structure

```
i18n/
├── config.ts           # Locale configuration + ENV variables
├── request.ts          # next-intl server configuration
├── index.ts            # Barrel exports
└── modules/            # Translation modules
    ├── common/         # Common translations (buttons, labels)
    ├── auth/           # Authentication translations
    ├── nav/            # Navigation translations
    ├── settings/       # Settings translations
    ├── errors/         # Error messages
    └── metadata/       # Page metadata
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

| Variable | Description | Default |
|----------|-------------|---------|
| `NEXT_PUBLIC_DEFAULT_LOCALE` | Default locale | `zh-Hans` |
| `NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED` | Show language switcher | `true` |

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
import { getT } from '@/i18n';

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

| Variable | Description | Default |
|----------|-------------|---------|
| `NEXT_PUBLIC_DEFAULT_LOCALE` | Default locale | `zh-Hans` |
| `NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED` | Show language switcher | `true` |

## Type Safety

- Base locale (`zh-Hans.ts`) defines the message structure
- Other locales must implement the same structure (enforced by TypeScript)
- Missing keys will cause compile-time errors
