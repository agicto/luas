# AI Agent Guide

> This document is designed for AI coding assistants (Cursor, Windsurf, GitHub Copilot, etc.) to understand and work with this codebase effectively.

## Project Overview

**LlamaFront AI Scaffold** - A production-ready Next.js scaffold optimized for rapid AI-assisted development.

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

## Directory Structure

```
src/
├── app/                    # Next.js App Router
│   ├── (auth)/             # Public auth route group (login, register)
│   ├── (normal)/           # Protected route group
│   │   └── console/        # Admin dashboard
│   ├── (site)/             # Public site route group
│   ├── api/                # API Route Handlers (Mock endpoints)
│   │   └── auth/           # Auth endpoints (Mock by default)
├── components/
│   ├── ui/                 # shadcn/ui primitives (DO NOT MODIFY)
│   └── features/           # Business feature components
├── config/                 # App configuration
├── constants/              # Route constants, enums
├── hooks/                  # Custom React hooks
├── http/                   # HTTP client (axios wrapper)
├── i18n/                   # Internationalization
│   ├── config.ts           # Locale config + ENV variables
│   ├── translations.ts     # Unified translation hooks (useT, getT)
│   └── modules/            # Per-module translations (common, auth, etc.)
├── providers/              # React context providers
├── services/               # API service layer (Zod validated)
├── store/                  # Zustand stores (dumb state only)
├── test/                   # Test utilities and setup (unit tests in src/test)
├── themes/                 # Design tokens (OKLCH, CSS variables)
├── types/                  # TypeScript type definitions
└── utils/                  # Pure utility functions
```

## Architecture Patterns

### 1. Mock API Architecture

All API calls go through Next.js Route Handlers under `/api/*`:

```
Browser → /api/auth/* → Mock handlers (Next.js API routes)
```

**Benefits:**
- No external dependencies for development
- httpOnly cookie sessions (secure)
- Fast local development without backend

### 2. Authentication Flow

**Token Modes (configured via `NEXT_PUBLIC_AUTH_TOKEN_MODE`):**

| Mode | Description |
|------|-------------|
| `basic` | Single access token, no refresh |
| `refresh` | Access + refresh token pair (default) |

**Auth Endpoints (Mock API):**

```
# Mock Backend (in src/app/api/auth)
POST /api/auth/login     → Mock login (admin@example.com / admin123)
POST /api/auth/register  → Mock user registration
GET  /api/auth/me        → Get current user
POST /api/auth/logout    → Clear cookies
GET  /api/auth/setup-status    → Get setup status
POST /api/auth/setup           → Initial system setup
GET  /api/auth/system-features → Get feature flags
```

**Route Groups:**
- `(auth)/*` - Public auth pages (login, register)
- `(normal)/*` - Protected routes (requires AuthGuard/Middleware)
- `(site)/*` - Public pages

**Protecting Routes:**

```typescript
// app/(normal)/layout.tsx
import { AuthGuard } from '@/components/auth-guard';

export default function NormalLayout({ children }) {
  return <AuthGuard>{children}</AuthGuard>;
}
```

### 3. State Management

- **Auth state**: `src/store/auth-store.ts` (Zustand + persist)
  - Stores user data, system features, and authentication status.
  - Contains state and setters; async initialization logic is included for convenience.
- **Auth actions**: `src/hooks/use-auth.ts` (React Query)
  - Handles `login`, `register`, and `logout`.
  - Includes built-in toast notifications and redirection.
  - Usage: `const { mutate: login } = useLogin();`
- **Server state**: React Query for all API data.
- **UI state**: `src/store/ui-store.ts` (Zustand).
- **Auth config**: `src/config/auth.ts` (token modes, routes).

### 4. Internationalization (i18n)

The project uses `next-intl` with a unified translation pattern that supports both dot notation and namespace-based access.

**Client Components:**
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

**Server Components:**
```tsx
import { getT } from '@/i18n';

export default async function Page() {
  const t = await getT();
  return <h1>{t('common.loading')}</h1>;
}
```

**Available Namespaces:** `common`, `auth`, `nav`, `settings`, `errors`, `metadata`

**Type Safety:** Invalid keys will cause TypeScript errors at compile time.

## Code Conventions

### Must Follow

- **Package manager**: `pnpm` only
- **Comments**: English only
- **Tests**: Place in `src/test` directory
- **Hot reload**: Do NOT restart dev server (auto-updates)

### 5. Theme System (Design Tokens & OKLCH)

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
import { useAuthStore } from '@/store/auth-store';
```

## Environment Variables

### Required

```bash
UPSTREAM_API_BASE=https://your-backend.com/api
```

### Optional

```bash
NEXT_PUBLIC_APP_URL=http://localhost:3000
NEXT_PUBLIC_API_URL=/api
NEXT_PUBLIC_GA_MEASUREMENT_ID=G-XXXXXXX

# Auth configuration
NEXT_PUBLIC_AUTH_TOKEN_MODE=refresh         # basic | refresh
```

## Common Tasks

### Adding a New Page

1. Create file in `src/app/(site)/page-name/page.tsx`
2. Add route to `src/constants/routes.ts`
3. Add translations to `src/i18n/modules/[module]/zh-Hans.ts` and other locales

### Adding a New API Endpoint

1. Create folder in `src/app/api/endpoint-name/`
2. Add `route.ts` with HTTP method handlers
3. Use `cookies()` from `next/headers` for auth if needed

### Adding a New Component

1. **UI primitives** → Use shadcn/ui: `npx shadcn@latest add [component]`. These always go in `src/components/ui/`.
2. **Feature components** → Create in `src/components/features/[module]/`. 
   - **CRITICAL**: Do NOT place components in `app/` (route) directories. All reusable or module-specific components MUST be centralized in `src/components/`.
   - Organise by functional module (e.g., `src/components/features/console/`).
3. **Common components** → Generic, non-business specific components should be in `src/components/common/`.

### Adding a New Data Module (Service-Hook-Type Pattern)

All data handling must follow the strict **Service-Hook-Type** layered architecture to ensure separation of concerns and type safety.

#### 1. Define Types (`src/types/*.ts`)
All data structures (Domain models, DTOs, Query schemas) must be strictly typed.

```typescript
export interface ExampleItem { id: string; title: string; status: 'active' | 'inactive'; }
export interface UpdateExampleRequest { title?: string; status?: 'active' | 'inactive'; }
```

#### 2. Implement Stateless Service (`src/services/*.ts`)
Services are pure functional objects.
- **Stateless**: They do not hold state or use hooks.
- **Reusable**: Must be usable in both Client and Server Components.
- **Simple**: Just wrappers around `request`.

```typescript
export const exampleService = {
  get: (id: string) => request.get<ExampleItem>(`/api/example/${id}`),
  update: (id: string, data: UpdateExampleRequest) => request.put<ExampleItem>(`/api/example/${id}`, data),
};
```

#### 3. Implement Encapsulated Hooks (`src/hooks/*.ts`)
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

## API Response Format

All BFF endpoints return consistent format:

```typescript
// Success
{ "data": { ... } }

// Error
{ "error": "message", "code": "ERROR_CODE" }
```

## Do NOT

- Modify files in `src/components/ui/` (shadcn/ui managed)
- Use `localStorage` for tokens (use httpOnly cookies)
- Add business-specific logic to scaffold (keep generic)
- Create test files in business modules (use `tests/`)
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
