---
name: i18n-handler
description: Change Luas next-intl messages or translation boundaries. Use for locale keys, namespaces, server/client translation APIs, or locale routing.
---

# Internationalization Handler

## Purpose

Change localized copy without weakening message type safety, request locale
resolution, or route-level client payload budgets.

Read [`src/i18n/README.md`](../../../src/i18n/README.md) only when adding a
locale/namespace or changing the architecture. Ordinary copy changes need the
owning locale files and focused tests only.

## Core Contract

- `en-US` is the canonical source schema.
- `zh-Hans` is a structurally checked translation locale.
- `MessageSchema` derives keys, scopes, and ICU argument names from source
  literals.
- Every locale preserves the source tree and ICU variable names.
- Locale IDs use BCP 47 tags from `src/i18n/locales.ts`.
- The configured default locale does not change schema ownership.

## API Choice

| Surface | API |
|---|---|
| Server Component or metadata | `getT` from `@/i18n/server` |
| Client Component | `useT` from `@/i18n` |
| Many keys in one subtree | Scoped `getT('scope')` or `useT('scope')` |
| A few cross-namespace keys | Dot notation such as `t('common.save')` |

Prefer server translation and pass final labels to interactive leaves when
that avoids adding a client namespace. Never import the server entry from a
Client Component.

## Message Change Workflow

1. Choose the feature-owned namespace; do not place feature copy in `common`.
2. Add semantic source keys to `<namespace>/en-US.ts`.
3. Add the aligned `zh-Hans.ts` values with identical ICU arguments.
4. Use complete messages instead of concatenating translated fragments.
5. Map stable backend `error_code` values to local copy; do not render backend
   `message` text directly.
6. Run the i18n type/copy check that owns the changed surface.

## Namespace Or Locale Changes

For a new namespace, update:

- `module-names.ts`;
- both locale files and the namespace `index.ts`;
- `loader.ts`;
- `modules/index.ts` schema and parity registration;
- the nearest client route scope only when browser interaction needs it.

For a new locale, also update locale resolution, environment validation,
DayPicker mapping, all formal message modules, and formatting tests. Do not
ship a partially translated formal scaffold surface without an explicit
downstream fallback policy.

## Client Message Budget

`CLIENT_MESSAGE_NAMESPACES` owns browser serialization. When a Client
Component needs another namespace:

1. add it to the nearest route scope;
2. select it in the owning server layout;
3. wrap the interactive subtree with `RouteMessagesProvider`;
4. update `src/test/i18n-client-messages.test.tsx`.

Never send the complete `getMessages()` tree through the root client provider.

## Formal Copy Boundary

Site, auth, console, metadata, and shared shell copy use translation APIs.
Exact brand names, technical identifiers, code literals, disposable devtools,
and the dependency-light global error fallback are narrow exceptions.

## Verification

```bash
corepack pnpm vitest run src/test/i18n-types.test.ts src/test/locale-resolution.test.ts
corepack pnpm lint:i18n-copy
corepack pnpm type-check
```

Run the client-message test or production build only when namespace delivery
or route payloads changed.

## Related Skills

- `api-error-handling`: error-code mapping behavior changes.
- `accessibility-audit`: dedicated language/RTL/WCAG review.
- `frontend-design`: explicit translation-length layout redesign.
