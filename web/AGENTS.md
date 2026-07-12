# AI Agent Guide

> This document is designed for AI coding assistants (Cursor, Windsurf, GitHub Copilot, etc.) to understand and work with this codebase effectively.

## Project Overview

**Luas** - A production-ready Next.js scaffold optimized for rapid AI-assisted development.

| Tech | Version | Purpose |
|------|---------|---------|
| Next.js | 16.x | App Router, RSC, API Routes |
| React | 19.x | UI Library |
| TypeScript | 5.x | Type Safety |
| Tailwind CSS | 4.x | Styling |
| shadcn/ui | Latest | UI Components |
| Zustand | 5.x | State Management |
| React Query | 5.x | Server State |
| next-intl | 4.x | i18n |

## AI Agent Skills

The web side ships with a Skills System in `.agents/skills/` that codex CLI auto-loads. Each skill is a self-contained workflow loaded only when its description matches the task.

| Skill | When to Use |
|---|---|
| [`vercel-react-best-practices`](./.agents/skills/vercel-react-best-practices/) | Writing or refactoring React / Next.js code |
| [`frontend-design`](./.agents/skills/frontend-design/) | Building components, pages, dashboards, landing pages |
| [`web-design-guidelines`](./.agents/skills/web-design-guidelines/) | Applying project design tokens and component rules |
| [`ui-styling-guide`](./.agents/skills/ui-styling-guide/) | Tailwind + shadcn styling patterns |
| [`data-state-management`](./.agents/skills/data-state-management/) | Zustand / React Query / form state |
| [`i18n-handler`](./.agents/skills/i18n-handler/) | next-intl translation keys and scoped translations |
| [`api-error-handling`](./.agents/skills/api-error-handling/) | Client error contracts and user-facing surfaces |
| [`environment-config`](./.agents/skills/environment-config/) | Env vars, runtime config, build-time vs runtime |
| [`webapp-testing`](./.agents/skills/webapp-testing/) | Browser-driven test patterns (Playwright) |
| [`testing-standards`](./.agents/skills/testing-standards/) | Unit / component test conventions |
| [`utility-tooling`](./.agents/skills/utility-tooling/) | Project utility patterns |
| [`project-strategy`](./.agents/skills/project-strategy/) | Long-term direction and trade-off framing |
| [`skill-creator`](./.agents/skills/skill-creator/) | Creating or updating a skill |
| [`accessibility-audit`](./.agents/skills/accessibility-audit/) | WCAG 2.2 AA review (keyboard, ARIA, contrast) |
| [`web-perf`](./.agents/skills/web-perf/) | Core Web Vitals audit (LCP / INP / CLS) for Next 16 |

Root-level skills under `../.agents/skills/` — `grill-before-build`, `systematic-debugging`, `verification-before-completion`, `pr-description-writer` — also apply here.

## Directory Structure

```
src/
├── app/                    # Next.js App Router
│   ├── (auth)/             # Public auth route group (login, register)
│   ├── (protected)/        # Protected route group
│   │   ├── (console)/      # Console pages
│   │   └── (devtools)/     # Internal demo/playground pages
│   ├── (site)/             # Public site route group
│   ├── api/                # Mock BFF route handlers
│   │   └── auth/           # Mock BFF auth endpoints
├── components/
│   ├── ui/                 # shadcn/ui primitives (DO NOT MODIFY)
│   ├── common/             # Shared layout/common components
│   └── features/           # Shared feature-facing UI blocks
├── features/               # Feature-first folders (preferred)
│   ├── auth/               # components, hooks, services, store, server, types
│   └── example/            # hooks, services, server, types
├── config/                 # App configuration
├── constants/              # Route constants, enums
├── hooks/                  # Shared generic hooks
├── http/                   # HTTP client (axios wrapper)
├── i18n/                   # Internationalization
│   ├── config.ts           # Locale config + ENV variables
│   ├── translations.ts     # Client translation hook (useT)
│   ├── server.ts           # Server translation accessor (getT)
│   └── modules/            # Translation namespaces (common, auth, etc.)
├── providers/              # React context providers
├── services/               # Compatibility exports for feature services
├── store/                  # Shared global stores only
├── test/                   # Test utilities and setup (unit tests in src/test)
├── themes/                 # Design tokens (OKLCH, CSS variables)
├── types/                  # Shared cross-feature type definitions
└── utils/                  # Pure utility functions
```

## Architecture Patterns

### 1. Mock BFF Architecture

Default development calls use `NEXT_PUBLIC_API_URL=/api`, so feature services go through
Next.js route handlers under `src/app/api/**`:

```
Browser -> src/http/request.ts -> /api/* -> Mock BFF route handlers
```

**Benefits:**
- No external dependencies for development
- httpOnly cookie sessions (secure)
- Fast local development without backend

For downstream production apps, point `NEXT_PUBLIC_API_URL` at the real API or a same-origin
proxy and keep the mock BFF disabled. See `docs/MOCK_BFF.md` for replacement and deletion steps.

### 2. Authentication Flow

**Auth Endpoints (Mock BFF):**

```
# Mock BFF (in src/app/api/auth)
POST /api/auth/login     → Mock login (admin@example.com / admin123)
POST /api/auth/register  → Mock user registration
GET  /api/auth/me        → Get current user
POST /api/auth/logout    → Clear cookies
```

**Route Groups:**
- `(auth)/*` - Public auth pages (login, register)
- `(protected)/*` - Protected routes (enforced by `middleware.ts` and `AuthGuard`)
- `(protected)/(console)/*` - Business console pages
- `(protected)/(devtools)/*` - Internal playground/demo pages
- `(site)/*` - Public pages

**Protecting Routes:**

```typescript
// middleware.ts
// Redirects unauthenticated traffic away from /console, /styleguide, and /i18n-test

// app/(protected)/layout.tsx
import { AuthGuard } from '@/features/auth';

export default function ProtectedLayout({ children }) {
  return <AuthGuard>{children}</AuthGuard>;
}
```

### 3. State Management

- **Auth state**: `src/features/auth/store/auth-store.ts` (Zustand)
  - Mirrors the current server session in memory.
  - Initializes via `/api/auth/me` on app startup.
- **Auth actions**: `src/features/auth/hooks/use-auth.ts` (React Query)
  - Handles `login`, `register`, and `logout`.
  - Includes built-in toast notifications and redirection.
  - Usage: `const { mutate: login } = useLogin();`
- **Server state**: React Query for all API data.
- **UI state**: `src/store/ui-store.ts` (Zustand).
- **Auth config**: `src/config/auth.ts` (cookie names, routes, demo account).

### 4. Provider Placement

Keep providers as low as the routes that need them:

- Root layout owns only app-wide client context such as theme and the `common` / `errors` client message namespaces; optional analytics stays server-rendered until `next/script` activates it.
- `(auth)` owns `QueryProvider` and `Toaster` because login/register forms use React Query mutations and toast feedback.
- `(auth)` keeps its visual shell server-rendered; only interactive leaves such as `LanguageSwitcher`, forms, and `QueryProvider` cross the client boundary.
- `(protected)` owns `AuthenticatedProviders` and `Toaster`; the provider combines `QueryProvider` and `AuthProvider` before `AuthGuard`.
- Auth, console, and devtool routes append only the client message namespaces declared in `src/i18n/client-message-namespaces.ts`; server translations still have the complete request message tree.
- Public `(site)` routes must not subscribe to `auth-store` or initialize mock auth on first load.
- If a new feature needs React Query, add `QueryProvider` at the nearest route group instead of moving it back to root.
- `src/test/public-route-boundary.test.ts` fails if public `(site)` routes pull in auth, query, HTTP, mock BFF, mock session, toast, or Zustand runtime dependencies.
- `src/test/i18n-runtime-boundary.test.ts` keeps the client/server translation entry points separate and prevents the auth shell from becoming a Client Component.
- `src/test/i18n-client-messages.test.tsx` guards global and route-owned client message scopes so unrelated namespaces are not serialized into every page.
- `src/test/root-runtime-boundary.test.ts` keeps toast and optional analytics out of the root client graph and requires Next.js `error.tsx` / `global-error.tsx` conventions instead of a custom catch-all wrapper.

### Error Boundaries

- `src/app/error.tsx` is the translated recovery UI for uncaught route-segment errors.
- `src/app/global-error.tsx` is the dependency-light fallback for root layout failures and must define its own `html` and `body`.
- Use nested `error.tsx` files only when a route group needs a genuinely different recovery workflow.
- Do not wrap the root layout in a custom Client Component error boundary; it expands the shared hydration graph and duplicates App Router behavior.

### 5. Internationalization (i18n)

The project uses `next-intl` with a unified translation pattern that supports both dot notation and namespace-based access.

#### Type System

The i18n system derives key and scope safety from the message tree through:
- **`AllTranslationKeys`**: Union of translatable leaf paths such as `auth.login`; object nodes are excluded.
- **`AllScopePaths`**: Union of valid object paths such as `test.level1`; leaf paths are excluded.
- **`ScopedTranslationKeys<P>`**: Relative leaf keys available below scope `P`.
- **`ScopedTranslations<P>`**: Translator whose key parameter is derived from `ScopedTranslationKeys<P>`.
- **`UnifiedTranslations`**: Combined type that supports both dot notation and namespace accessors

#### Client Components
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

Client Components only receive explicitly owned namespaces. When a route introduces a new client-side `useT` namespace, add it to `src/i18n/client-message-namespaces.ts` and mount the selected messages through `RouteMessagesProvider` at the nearest owning route layout. Do not restore the full `getMessages()` object to the root client provider.

#### Server Components
```tsx
import { getT } from '@/i18n/server';

export default async function Page() {
  const t = await getT();
  return <h1>{t('common.loading')}</h1>;
}
```

#### Scoped Translations
For components with many translations from a single namespace:
```tsx
function SettingsPage() {
  const t = useT('settings');  // Scoped to 'settings' namespace
  return <h1>{t('title')}</h1>;  // settings.title
}
```

#### Available Translation Namespaces

| Namespace | Description |
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

#### Key Files

| File | Purpose |
|------|---------|
| `src/i18n/config.ts` | Locale configuration and settings |
| `src/i18n/translations.ts` | Client-only `useT` implementation |
| `src/i18n/server.ts` | Server-only `getT` implementation |
| `src/i18n/translation-shared.ts` | Message-tree-derived key/scope types and pure facade construction |
| `src/i18n/module-names.ts` | Canonical translation module names |
| `src/i18n/loader.ts` | Dynamic translation namespace loading |
| `src/i18n/client-message-namespaces.ts` | Global and route-owned client namespace registry |
| `src/i18n/message-selection.ts` | Type-safe top-level namespace selection |
| `src/i18n/route-messages-provider.tsx` | Additive nested client message provider |
| `src/i18n/modules/index.ts` | Static type generation from translation modules |
| `src/i18n/README.md` | Detailed documentation with examples |

**Type Safety:** Invalid keys will cause TypeScript errors at compile time.
`src/test/i18n-types.test.ts` guards valid scopes, leaf-only global keys, relative scoped keys, and runtime prefix composition.

#### Core Copy Boundary

- Root metadata, `(site)`, `(auth)`, `(protected)/(console)`, and their shared shell components must resolve user-facing copy through `getT` or `useT`.
- Prefer `getT` in Server Components. Add a client namespace only when an interactive leaf calls `useT`.
- Exact brand names, technical identifiers, and text inside `<code>` may remain literals when translating them would be incorrect.
- `devtools` and `example` are disposable scaffold surfaces and are outside the core copy guard. They must not leak into formal site, auth, or console copy.
- `global-error.tsx` remains a dependency-light root fallback. UI primitive defaults are governed through their component APIs and should receive translated labels from formal callers.
- `pnpm lint:i18n-copy` enforces this boundary, and `pnpm lint` runs it automatically.

## Code Conventions

### Must Follow

- **Package manager**: `pnpm` only
- **Comments**: English only
- **Tests**: Place in `src/test` directory
- **Hot reload**: Do NOT restart dev server (auto-updates)

### 6. Theme System (Design Tokens & OKLCH)

The project uses a structured Design Token system based on OKLCH and CSS variables, layered for better governance and maintainability.

#### Layered Architecture:
1.  **Primitives (`src/themes/primitives.css`)**: Base color palette (e.g., `--neutral-500`, `--blue-500`). **Do not use directly.**
2.  **Semantic Tokens (`light.css` / `dark.css`)**: Functional naming based on purpose.
    - **Backgrounds**: `bg-canvas`, `bg-surface`, `bg-subtle`.
    - **Foregrounds**: `text-main`, `text-subtle`, `text-muted`.
    - **Borders**: `border-main`, `border-subtle`, `border-strong`.
    - **Brand**: `brand-main`, `brand-subtle`, `brand-strong`.
    - **States**: `success`, `warning`, `error`, `info`, `highlight`.

#### Usage in Tailwind (Strict Mode):
Always prefer semantic classes over raw color values.
```tsx
// Using Semantic Tokens
<div className="bg-bg-surface text-text-main border-border-subtle shadow-md p-4 rounded-lg">
  <h1 className="text-brand">Heading</h1>
  <p className="text-text-subtle">Description using subtle text.</p>
</div>

// Opacity modifiers work on all semantic tokens
<div className="border-brand/60 bg-brand-subtle/50">
  Integrated opacity support
</div>
```

### File Naming

- Components: `kebab-case.tsx` (e.g., `login-form.tsx`)
- Hooks: `use-*.ts` (e.g., `use-mobile.ts`)
- Types: `*.ts` in `types/` directory
- Utils: `*.ts` in `utils/` directory

### Import Order

```typescript
// 1. React/Next imports
import { useState } from 'react';
import Link from 'next/link';

// 2. Third-party libraries
import { useQuery } from '@tanstack/react-query';

// 3. Internal imports (absolute paths)
import { Button } from '@/components/ui/button';
import { useAuthStore } from '@/features/auth/store/auth-store';
```

## Environment Variables

The project uses a **Strict Environment Variable** system to ensure type safety and prevent runtime errors.

### 1. Explicit Runtime Entries

- Browser-safe values MUST be resolved without a schema-library runtime in `src/config/env.ts` and covered by the server-only schema in `src/config/env-validation.ts`.
- Secrets and server-only runtime values MUST be defined in `src/config/server-env.ts`, which is protected by `server-only`.
- Never re-export `server-env` from `src/config/index.ts` or another client-reachable barrel.

**❌ DO NOT:**
```typescript
const apiUrl = process.env.NEXT_PUBLIC_API_URL; // Unsafe, untyped
```

**✅ DO:**
```typescript
import { env } from '@/config/env';

const apiUrl = env.NEXT_PUBLIC_API_URL; // Typed, validated
```

Server modules use the explicit server entry:

```typescript
import { serverEnv } from '@/config/server-env';

const mockEnabled = serverEnv.MOCK_BFF_ENABLED;
```

This is enforced by `src/test/env-contract.test.ts`: production source files must not read `process.env` outside the environment entries, server values cannot leak into client config, and production mock BFF opt-in requires a strong `SESSION_SECRET`.

### 2. Validation (Zod)
We use `zod` to validate environment variables at runtime. If a required runtime variable is missing, the app fails to start with a clear error message.

### 3. Supported Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXT_PUBLIC_API_URL` | No | `/api` | Base URL for API requests. |
| `NEXT_PUBLIC_APP_URL` | No | `http://localhost:3000` | Absolute site URL for metadata, sitemap, and robots. |
| `NEXT_PUBLIC_DEFAULT_LOCALE` | No | `zh-Hans` | Default locale. Must be one of `locales` from `src/i18n/locales.ts`. |
| `NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED` | No | `true` | Shows or hides the language switcher. |
| `NEXT_PUBLIC_GA_MEASUREMENT_ID` | No | - | Google Analytics ID. |
| `NODE_ENV` | No | `development` | App environment (`development` \| `production` \| `test`). |
| `MOCK_BFF_ENABLED` | Production opt-in only | `false` | Enables development mock BFF route handlers in production runtime. Keep false for downstream production apps. |
| `SESSION_SECRET` | Production mock BFF | - | Server-only secret used to HMAC-sign the mock auth session cookie. Required when `MOCK_BFF_ENABLED=true`; use at least 32 characters. |

To add a new variable:
1. Add it to `.env.local`
2. Put browser-safe values in `src/config/env.ts` and secrets/server-only values in `src/config/server-env.ts`
3. Add the value to the matching Zod schema and exported object


## Tooling & Utility Standards (ARW)

To maintain a lean and consistent codebase, AI agents MUST follow the **"Search First"** rule before implementing any new utility or hook.

### 1. The "Search First" Rule
Before writing a new utility function or React hook, you **MUST**:
1.  **Check `src/utils/index.ts`**: Scan the exports to see if a similar utility already exists.
2.  **Check `src/hooks/`**: Browse the file names and signatures for existing logic.
3.  **Check Approved Libraries**: Verify if the functionality is provided by:
    - `date-fns`: For all date manipulations.
    - `lodash-es`: For complex object/array operations (use sparingly, prefer native).
    - `validator`: For complex string validation.

### 2. Implementation Priority
Follow this order of preference:
1.  **Native Web APIs**: `Intl`, `URL`, `Crypto`, etc.
2.  **Existing Project Utils/Hooks**: Reuse what's already in `src/utils` or `src/hooks`.
3.  **Approved Third-Party Libraries**: Use existing dependencies from `package.json`.
4.  **Custom Implementation**: Only if the above options are exhausted.

### 3. Utility & Hook Discovery Tags
Use these tags in JSDoc headers to aid AI discovery:
- `@util`: Marks a pure utility function.
- `@hook`: Marks a reusable React hook.

### 4. Contract for New Additions
- **Utils**: Must be pure functions in `src/utils/[category].ts`, exported via `index.ts`, with tests in `__tests__/`.
- **Hooks**: Must be in `src/hooks/use-[purpose].ts` and follow React Hook rules.

## Common Tasks

### Adding a New Page

1. Create file in `src/app/(site)/page-name/page.tsx`
2. Add route to `src/constants/routes.ts`
3. Add translations to `src/i18n/modules/[namespace]/zh-Hans.ts` and other locales

### Adding a New Mock BFF Route

1. Create folder in `src/app/api/endpoint-name/`
2. Add `route.ts` with HTTP method handlers
3. Call `guardMockBffRoute()` before reading request bodies or touching mock state
4. Return the shared contract shape from `contracts/README.md`
5. Use `cookies()` from `next/headers` for mock auth if needed

### Adding a New Component

1. **UI primitives** → Use shadcn/ui: `npx shadcn@latest add [component]`. These always go in `src/components/ui/`.
2. **Feature components** → Prefer `src/features/[feature]/components/`.
   - **CRITICAL**: Do NOT place reusable components in `app/` route directories.
   - Keep each feature's hooks, services, store, server helpers, and types inside the same `src/features/[feature]/` folder.
   - Use `src/components/features/` only for shared cross-feature UI blocks.
3. **Common components** → Generic, non-business specific components should be in `src/components/common/`.

#### Atomic Component Contract

To ensure engineering rigor and performance, all components MUST follow these rules:

- **Named Exports**: Always use named exports. Do NOT use `export default`.
  - `export function ComponentName({ ... }: ComponentNameProps) { ... }`
- **Strict Typing**: Always define an interface for props named `[ComponentName]Props`.
- **RSC First Strategy**: 
  - Components are **Server Components** by default.
  - If a component needs interactivity (hooks or events), extract the interactive logic into a **small, leaf-level Client Component** (marked with `'use client'`). 
  - Avoid turning large feature components into Client Components.
- **Localized Core Copy**: User-facing copy in formal site, auth, and console surfaces must use `getT` or `useT`; follow the Core Copy Boundary above for narrow exceptions and disposable surfaces.
- **Icon Consistency**: Use `lucide-react`. Standardize size using Tailwind's `size-4` (16px) or `size-5` (20px) for consistent alignment.

#### Performance Optimization Rules

Following [Vercel React Best Practices](./.agents/skills/vercel-react-best-practices/SKILL.md) for optimal performance:

- **Dynamic Imports** (`bundle-dynamic-imports`): Use `next/dynamic` for components > 50KB (charts, editors, rich-text, maps):
  ```tsx
  const HeavyEditor = dynamic(() => import('./heavy-editor'), {
    loading: () => <Skeleton className="h-64" />,
  });
  ```

- **React.memo** (`rerender-memo`): Use for expensive child components that receive stable props:
  ```tsx
  export const ExpensiveList = React.memo(function ExpensiveList({ items }: Props) {
    // Heavy rendering logic
  });
  ```

- **Conditional Rendering** (`rendering-conditional-render`): Use ternary operators, not `&&`:
  ```tsx
  // ✅ Correct
  {condition ? <Component /> : null}
  
  // ❌ Avoid (may render '0' or 'false')
  {condition && <Component />}
  ```

- **RSC Caching** (`server-cache-react`): Use `React.cache()` for per-request deduplication in Server Components:
  ```tsx
  import { cache } from 'react';
  
  export const getUser = cache(async (id: string) => {
    return await db.user.findUnique({ where: { id } });
  });
  ```

#### Component Annotation Standard (CAS)

All components MUST include a standardized JSDoc header for discovery and AI-assisted reuse:

```typescript
/**
 * @component [Formal Name]
 * @category [UI | Feature | Common]
 * @status [Stable | Beta | Experimental]
 * @description [Concise purpose]
 * @usage [When/How to use]
 * @example
 * <ComponentName prop={value} />
 */
```

### Adding Feature Data Flow (Service-Hook-Type Pattern)

All data handling must follow the strict **Service-Hook-Type** layered architecture to ensure separation of concerns and type safety.

#### 1. Define Types (`src/features/[feature]/types.ts`)
All data structures (Domain models, DTOs, Query schemas) must be strictly typed.

```typescript
export interface ExampleItem { id: string; title: string; status: 'active' | 'inactive'; }
export interface UpdateExampleRequest { title?: string; status?: 'active' | 'inactive'; }
```

#### 2. Implement Stateless Service (`src/features/[feature]/services/*.ts`)
Services are pure functional objects.
- **Stateless**: They do not hold state or use hooks.
- **Dedicated Clients**: Use the appropriate HTTP client for the feature.
- **Simple**: Just wrappers around named `request` instances.

**Standard Pattern: Named Request Instances**
Define specialized clients in `src/http/request.ts` to manage multiple base URLs (e.g., standard API vs. file service).

```typescript
// 1. Define instances in src/http/request.ts
export const request = createRequest({ baseURL: env.API_URL }); // Default
export const fileRequest = createRequest({ baseURL: env.FILE_URL, timeout: 60000 }); // Specialized

// 2. Map services to the correct client
// src/features/user/services/user-service.ts
import request from '@/http/request'; 
export const userService = { get: () => request.get('/user') };

// src/features/file/services/file-service.ts
import { fileRequest } from '@/http/request';
export const fileService = { upload: (file) => fileRequest.post('/upload', file) };
```

#### 3. Implement Encapsulated Hooks (`src/features/[feature]/hooks/*.ts`)
Hooks manage React Query state and side effects.

**Standard Pattern: Full CRUD Optimistic Updates**
Provide immediate UI feedback and handle errors via **Refetch-on-Failure**.

```typescript
// 1. Key Factory
export const exampleKeys = {
  all: ['examples'] as const,
  lists: () => [...exampleKeys.all, 'list'] as const,
  detail: (id: string) => [...exampleKeys.all, 'detail', id] as const,
};

// 2. Query Hooks
export const useExample = (id: string) => useQuery({
  queryKey: exampleKeys.detail(id),
  queryFn: () => exampleService.get(id),
});

// 3. Mutation Hooks (Optimistic Template)
export function useUpdateExample() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string, data: UpdateExampleRequest }) => 
      exampleService.update(id, data),
    
    // Step 1: Push optimistic update to UI
    onMutate: async ({ id, data }) => {
      await queryClient.cancelQueries({ queryKey: exampleKeys.detail(id) });
      const prev = queryClient.getQueryData(exampleKeys.detail(id));
      if (prev) queryClient.setQueryData(exampleKeys.detail(id), { ...prev as any, ...data });
      return { prev };
    },
    
    // Step 2: Rollback via Refetch on failure
    onError: (err, { id }) => {
      queryClient.invalidateQueries({ queryKey: exampleKeys.detail(id) });
      toast.error(err.message);
    },
    
    // Step 3: Final synchronization
    onSettled: (data, err, { id }) => {
      queryClient.invalidateQueries({ queryKey: exampleKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: exampleKeys.lists() });
    }
  });
}
```

**Key Strategies:**
- **Cancel outgoing refetches** in `onMutate` to prevent race conditions.
- **Rollback via Invalidation**: Simpler and more robust than manual state restoration.
- **Always Sync**: Final invalidation in `onSettled` ensures local state perfect alignment with the server.

## API Error Handling

The project uses a standardized error code system to ensure consistency between the frontend and backend.

### 1. Error Response Format
All error responses from the backend (BFF or mock) MUST follow this format:
```json
{
  "code": 404,
  "error_code": "COMMON.NOT_FOUND",
  "message": "Human-readable error message",
  "request_id": "req_123"
}
```

### 2. Error Code Namespaces
Backend `error_code` is the canonical branching field. New backend, BFF, and mock responses MUST use uppercase dot-separated values from `ApiErrorCode`.

| Namespace | Owner | Examples |
|-----------|-------|----------|
| `COMMON.*` | Shared API contract | `COMMON.NOT_FOUND`, `COMMON.VALIDATION_FAILED` |
| `AUTH.*` | Auth API contract | `AUTH.UNAUTHORIZED`, `AUTH.INVALID_CREDENTIALS` |
| `USER.*` | User API contract | `USER.EMAIL_ALREADY_EXISTS` |
| `API_KEY.*` | API key API contract | `API_KEY.REVOKED` |
| `CLIENT.*` | Frontend-only fallback | `CLIENT.NETWORK_ERROR`, `CLIENT.TIMEOUT` |

Legacy frontend-only codes such as `VAL_400` may be normalized for backward compatibility, but new code MUST NOT emit them.

### 3. Usage in Code
- **Constants**: Use `ApiErrorCode` from `@/http/codes` for backend `error_code` values; use `ClientErrorCode` only when no backend response exists. This split is enforced by `src/test/error-code-vocabulary.test.ts`.
- **Mock routes**: Return `{ code, error_code, message, request_id? }`; do not return legacy `{ error, code: "VAL_400" }` shapes.
- **Mock BFF guard**: New mock route handlers must call `guardMockBffRoute()` from `@/app/api/_shared/mock-bff` before reading the request body or touching mock state. This is enforced by `src/test/mock-bff-route-contract.test.ts`.
- **Validation**: Return `400 COMMON.INVALID_INPUT` for malformed JSON or transport-level input errors; return `422 COMMON.VALIDATION_FAILED` with `errors` for schema/field validation failures.
- **Handling**: Errors are automatically caught by `HttpClient` and passed to `handleError`. Use `skipErrorHandler: true` in request config to handle errors manually in components or hooks.

## API Response Format

## Do NOT

- Modify files in `src/components/ui/` (shadcn/ui managed)
- Use `localStorage` for tokens (use httpOnly cookies)
- Add business-specific logic to scaffold (keep generic)
- Bury shared contract tests in feature folders; use `src/test` for cross-feature or contract checks.
- Use Chinese in code comments
- Generate `.sh` or `.md` files unless explicitly requested

## Quick Reference

| Action | Command |
|--------|---------|
| Install deps | `pnpm install` |
| Dev server | `pnpm dev` |
| Build | `pnpm build` |
| Type check | `pnpm type-check` |
| Lint | `pnpm lint` |
| Format | `pnpm format` |
| Add UI component | `npx shadcn@latest add [name]` |
