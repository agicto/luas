---
name: i18n-handler
description: Guidelines for managing internationalization (i18n) in the project using next-intl and unified translation patterns.
---

# i18n-handler

## Overview

This skill provides comprehensive instructions for implementing and maintaining internationalization (i18n) within the LlamaFront AI Scaffold. It covers client and server component usage, type-safe translation keys, and module organization.

## Guidelines

### 1. Unified Translation Pattern
The project uses `next-intl` with a unified pattern that supports both dot notation (recommended) and namespace-based access.

#### Client Components
Use the `useT` hook for translations in client-side components.

```tsx
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
import { getT } from '@/i18n';

export default async function Page() {
  const t = await getT();
  return <h1>{t('common.loading')}</h1>;
}
```

### 2. Module Organization
Translations are organized into functional modules located in `src/i18n/modules/[module]/`.

- **Available Namespaces**: `common`, `auth`, `nav`, `settings`, `errors`, `metadata`
- **Supported Locales**: Configured in `src/i18n/config.ts`.

### 3. Adding New Translations
When adding a new page or feature:
1. Identify the relevant module in `src/i18n/modules/`.
2. Add the translation key and its values to `zh-Hans.ts` and other supported locale files.
3. Ensure the keys are consistent across all locale files to maintain type safety.

## Usage Scenarios

| Task | Action |
|------|--------|
| New Page | Add translations to `src/i18n/modules/[module]/[locale].ts` and update `src/constants/routes.ts`. |
| New Component | Use `useT` (Client) or `getT` (Server) for all user-facing text. |
| Error Messages | Use the `errors` namespace and follow the standardized error handling patterns. |

> [!IMPORTANT]
> **Zero Hardcoded Strings**: All user-facing text MUST use i18n hooks. Never hardcode text directly in JSX.
