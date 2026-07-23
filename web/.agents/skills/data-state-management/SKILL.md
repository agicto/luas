---
name: data-state-management
description: Implement Luas feature data and client state. Use for TanStack Query, Zustand, service-hook-type flow, cache invalidation, or optimistic updates.
---

# data-state-management

## Overview

This skill outlines the state management architecture of the project. It focuses on the separation of concerns between stateless services, React Query hooks for server state, and Zustand for global UI state.

## Guidelines

### 1. State Categories
- **Auth State**: `src/features/auth/store/auth-store.ts` (provider-owned Zustand vanilla store).
  It mirrors the current session using one status value and is never persisted in browser storage.
  `unauthenticated`, `forbidden`, and `unavailable` are distinct facts; never collapse transport
  failure into logout state.
- **UI State**: `src/store/ui-store.ts` (Zustand). Temporary/global UI states.
- **Server State**: Managed by React Query for all API-driven data.

### 2. Service-Hook-Type Pattern
All data handling must follow this layered architecture:

#### Layer 1: Types (`src/types/*.ts`)
Strictly type all domain models, DTOs, and query schemas.

#### Layer 2: Stateless Services (`src/services/*.ts`)
Pure functional objects that wrap HTTP requests. They do not hold state or use hooks.

```typescript
import request from '@/http/request'; 
export const userService = {
  get: () => request.get('/user'),
  update: (id: string, data: UserUpdate) => request.patch(`/user/${id}`, data)
};
```

#### Layer 3: Encapsulated Hooks (`src/hooks/*.ts`)
Manage React Query state and side effects. Implement full CRUD with optimistic updates.

```typescript
export function useUpdateExample() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }) => exampleService.update(id, data),
    meta: LOCAL_ERROR_HANDLING_META,
    onMutate: async ({ id, data }) => {
      await queryClient.cancelQueries({ queryKey: exampleKeys.detail(id) });
      const prev = queryClient.getQueryData(exampleKeys.detail(id));
      if (prev) queryClient.setQueryData(exampleKeys.detail(id), { ...prev, ...data });
      return { prev };
    },
    onError: (err, { id }, context) => {
      queryClient.setQueryData(exampleKeys.detail(id), context.prev);
      toast.error(t.errors('serverError'));
    },
    onSettled: (data, err, { id }) => {
      queryClient.invalidateQueries({ queryKey: exampleKeys.detail(id) });
    }
  });
}
```

### 3. Key Strategies
- **Optimistic Updates**: Provide immediate feedback.
- **Refetch-on-Failure**: Roll back state and refetch to ensure alignment.
- **Key Factories**: Use constant key objects for all query keys.
- **Request Isolation**: Create auth stores inside `AuthProvider`; never hydrate a module-level
  singleton with request-specific Server Component data.
- **Query Cache Isolation**: Create one QueryClient inside each `QueryProvider`; never export a
  module-level cache singleton across request-owned provider trees.
- **Error Ownership**: Set `meta: LOCAL_ERROR_HANDLING_META` whenever a hook or form presents its
  own failure. This prevents the global cache fallback from creating duplicate toasts.
- **Write Retry Safety**: Mutations do not retry by default. Opt in only when the endpoint contract
  provides idempotency evidence and the user workflow tolerates replay.
- **Explicit Bootstrap**: Protected Server Components pass `AuthBootstrap`. Definitive mock
  sessions start ready; `client-required` performs one deduplicated `/auth/me` request.
- **Failure Semantics**: `401` resolves as `unauthenticated`, `403` as `forbidden`, and network,
  timeout, rate-limit, `5xx`, malformed, or unknown failures as retryable `unavailable`.
- **Runtime Validation**: Receive auth service responses as `unknown`, then validate session and
  mutation JSON with the endpoint guards in `auth-response.ts`; compile-time DTO types are not
  evidence about network payloads.
- **Mutation Recovery**: Keep login/register failures in the form that can recover from them. Treat
  logout `401` as an idempotent completion, but preserve local auth state when availability or
  successful-response validity is unknown.

> [!TIP]
> **Stateless Services**: Always use the appropriate request instance (e.g., `request` vs `fileRequest`) from `src/http/request.ts`.

## Related Skills

Select another skill only when its distinct concern is active.

- [`api-error-handling`](../api-error-handling/): Error surfaces in queries and mutations.
- [`vercel-react-best-practices`](../vercel-react-best-practices/): Memoization patterns that interact with derived state.
- [`testing-standards`](../testing-standards/): Mocking and asserting state behavior in tests.
