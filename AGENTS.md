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

```
Login → POST /api/auth/login → Sets httpOnly cookies
        └── scaffold_access_token
        └── scaffold_refresh_token

Logout → POST /api/auth/logout → Clears cookies

Session Check → GET /api/auth/me → Returns user or 401
```

### 3. State Management

- **Auth state**: `src/store/auth-store.ts` (Zustand + persist)
- **Server state**: React Query for API data
- **UI state**: `src/store/ui-store.ts` (Zustand)

## Code Conventions

### Must Follow

- **Package manager**: `pnpm` only
- **Comments**: English only
- **Tests**: Place in `tests/` directory at project root
- **Hot reload**: Do NOT restart dev server (auto-updates)

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

1. Create `src/services/[name].ts` using functional API pattern:

```typescript
// services/example.ts
import { request } from '@/http';

const ENDPOINTS = {
  LIST: '/backend/api/examples',
  DETAIL: (id: string) => `/backend/api/examples/${id}`,
} as const;

export const exampleApi = {
  list: (params?: { page?: number }) =>
    request.get<Example[]>(ENDPOINTS.LIST, { params }),
  get: (id: string) =>
    request.get<Example>(ENDPOINTS.DETAIL(id)),
  create: (data: CreateExampleDto) =>
    request.post<Example>(ENDPOINTS.LIST, data),
  update: (id: string, data: UpdateExampleDto) =>
    request.patch<Example>(ENDPOINTS.DETAIL(id), data),
  delete: (id: string) =>
    request.delete<void>(ENDPOINTS.DETAIL(id)),
} as const;
```

2. Export from `src/services/index.ts`
3. Create hooks in `src/hooks/use-[name].ts` that call the service

### Adding Query Hooks

Use query key factory pattern:

```typescript
// hooks/use-examples.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { exampleApi } from '@/services';

export const exampleKeys = {
  all: ['examples'] as const,
  lists: () => [...exampleKeys.all, 'list'] as const,
  detail: (id: string) => [...exampleKeys.all, 'detail', id] as const,
};

export function useExamples() {
  return useQuery({
    queryKey: exampleKeys.lists(),
    queryFn: () => exampleApi.list(),
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
