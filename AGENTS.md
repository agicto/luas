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
│   ├── (auth)/             # Auth route group (login, register)
│   ├── (site)/             # Public site route group
│   ├── api/                # API Route Handlers (BFF layer)
│   │   ├── _lib/           # Shared API utilities
│   │   ├── auth/           # Auth endpoints
│   │   └── backend/        # Proxy to upstream API
│   └── console/            # Admin dashboard
├── components/
│   ├── ui/                 # shadcn/ui primitives (DO NOT MODIFY)
│   └── features/           # Business feature components
├── config/                 # App configuration
├── constants/              # Route constants, enums
├── hooks/                  # Custom React hooks
├── http/                   # HTTP client (axios wrapper)
├── i18n/                   # Internationalization (TypeScript modules)
│   ├── config.ts           # Locale config + ENV variables
│   └── modules/            # Per-module translations (common, auth, etc.)
├── providers/              # React context providers
├── services/               # API service layer
├── store/                  # Zustand stores
├── types/                  # TypeScript type definitions
└── utils/                  # Pure utility functions
```

## Architecture Patterns

### 1. BFF (Backend For Frontend)

All API calls go through Next.js Route Handlers under `/api/*`:

```
Browser → /api/auth/login → Upstream API
Browser → /api/backend/* → Upstream API (proxy)
```

**Benefits:**
- No CORS issues (same-origin)
- httpOnly cookie sessions (secure)
- Hide upstream API from client

### 2. Authentication Flow

**Token Modes (configured via `NEXT_PUBLIC_AUTH_TOKEN_MODE`):**

| Mode | Description |
|------|-------------|
| `basic` | Single access token, no refresh |
| `refresh` | Access + refresh token pair (default) |

**Auth Endpoints (Mock API):**

```
# Mock Backend (using Next.js API routes)
POST /api/auth/login → Mock login (admin@example.com / admin123)
POST /api/auth/refresh → Refresh tokens
GET  /api/auth/me → Get current user
POST /api/auth/logout → Clear cookies
```

**Route Groups:**
- `(auth)/*` - Public auth pages (login, register)
- `(normal)/*` - Protected routes (requires authentication)
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
  - **Dumb store**: Only contains state and simple setters.
- **Auth actions**: `src/hooks/use-auth.ts` (React Query)
  - Handles `login`, `register`, and `logout`.
  - Includes built-in toast notifications and redirection.
  - Usage: `const { mutate: login } = useLogin();`
- **Server state**: React Query for all API data.
- **UI state**: `src/store/ui-store.ts` (Zustand).
- **Auth config**: `src/config/auth.ts` (token modes, routes).

## Code Conventions

### Must Follow

- **Package manager**: `pnpm` only
- **Comments**: English only
- **Tests**: Place in `tests/` directory at project root
- **Hot reload**: Do NOT restart dev server (auto-updates)

### 4. Theme System (OKLCH)

The project uses a modular theme system based on OKLCH and CSS variables.

- **Storage**: `src/themes/` contains `light.css`, `dark.css`, and `index.css`.
- **Primary Tokens**: `primary`, `secondary`, `destructive`.
- **State Tokens**: `success`, `warning`, `info`.
- **Semantic Variations**:
  - `-subtle`: 10-15% opacity version for backgrounds.
  - `-strong`: Higher contrast version for borders or active states.
  - `-deeper`: Even higher contrast/darker version.
- **Surface Levels**:
  - `surface-1`: Base background.
  - `surface-2`: Elevated surface (card).
  - `surface-3`: Highest elevation (popover/modal).
- **Shadows**: Theme-aware shadows (`shadow-sm` to `shadow-2xl`) with dynamic color.

**Usage in Tailwind**:
```tsx
<div className="bg-primary-subtle text-primary-strong border-border-strong border-x-primary/60 shadow-md p-4 rounded-lg">
  {/* Content */}
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
NEXT_PUBLIC_USE_MOCK_AUTH=true              # Use mock auth (dev without backend)
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
3. Use `getAccessTokenFromCookies()` for auth

### Adding a New Component

1. UI primitives → Use shadcn/ui: `npx shadcn@latest add [component]`
2. Feature components → Create in `src/components/features/[feature]/`

### Adding a New Service

1. Create `src/services/[name].ts` using **Contract-First pattern**:

```typescript
import { z } from 'zod';
import { request } from '@/http';
import { components } from '@/types/api.generated'; // Generated via pnpm gen:api

// 1. Reference types from generated schema (Single Source of Truth)
export type Entity = components['schemas']['Entity'];
export type CreateEntityDto = components['schemas']['CreateEntityDto'];

// 2. Define Zod schemas to match the contract (Runtime validation)
export const EntitySchema = z.object({
  id: z.string(),
  name: z.string(),
  // ... other fields
});

// 3. Define endpoints
const ENDPOINTS = {
  BASE: '/backend/api/entities',
  DETAIL: (id: string) => `/backend/api/entities/${id}`,
} as const;

// 4. Create API with Zod validation
export const entityApi = {
  list: async (params?: any) => {
    const response = await request.get(ENDPOINTS.BASE, { params });
    return z.object({ data: z.array(EntitySchema), total: z.number() }).parse(response);
  },
  get: async (id: string) => {
    const response = await request.get(ENDPOINTS.DETAIL(id));
    return EntitySchema.parse(response);
  },
  create: async (data: CreateEntityDto) => {
    const response = await request.post(ENDPOINTS.BASE, data);
    return EntitySchema.parse(response);
  },
  delete: async (id: string) => {
    await request.delete(ENDPOINTS.DETAIL(id));
  },
} as const;
```

**Zod Validation Strategy (Selective):**

| Scenario | Validate? |
|----------|-----------|
| List/Detail APIs (user-facing data) | ✅ Yes |
| Create/Update response | ✅ Yes |
| Delete (returns void) | ❌ No |
| Internal data not directly displayed | ❌ Optional |

**Key Rules:**
- **Validate critical interfaces** - list, detail, create, update responses
- **Infer types from schemas** - `z.infer<typeof Schema>` for single source of truth
- **No schemas for request DTOs** - only validate incoming data, not outgoing
- **Skip validation for simple operations** - delete, toggle, etc.

2. Export from `src/services/index.ts`
3. Create hooks in `src/hooks/use-[name].ts` that call the service

### Adding Query Hooks

Use query key factory and optimistic update patterns:

```typescript
// hooks/use-entities.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { entityApi } from '@/services';
import { toast } from 'sonner';

export const entityKeys = {
  all: ['entities'] as const,
  lists: () => [...entityKeys.all, 'list'] as const,
  detail: (id: string) => [...entityKeys.all, 'detail', id] as const,
};

export function useEntities() {
  return useQuery({
    queryKey: entityKeys.lists(),
    queryFn: () => entityApi.list(),
  });
}

export function useCreateEntity() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data) => entityApi.create(data),
    onMutate: async (newData) => {
      await queryClient.cancelQueries({ queryKey: entityKeys.lists() });
      const previous = queryClient.getQueryData(entityKeys.lists());
      // Apply optimistic update to cache...
      return { previous };
    },
    onError: (err, _, context) => {
      queryClient.setQueryData(entityKeys.lists(), context?.previous);
      // NOTE: Global error handler handles toast automatically!
    },
    onSuccess: () => {
      toast.success('Created successfully');
      queryClient.invalidateQueries({ queryKey: entityKeys.lists() });
    },
  });
}
```

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
