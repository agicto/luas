---
name: project-strategy
description: High-level overview of the web shell, mock BFF architecture, and authentication flow.
---

# project-strategy

## Overview

This skill provides a high-level strategic overview of the Luas web shell. It covers the route structure, mock BFF architecture, and authentication flow so developers understand the core foundations of the web half.

## Project Structure

The project follows a modular structure under `src/`:
- `app/`: Next.js App Router route groups, pages, and mock BFF route handlers.
- `features/`: Feature-first folders with components, hooks, services, store, server helpers, and types.
- `components/`: shadcn/ui primitives (`ui/`), shared feature-facing UI blocks (`features/`), and common components (`common/`).
- `services/`: Compatibility exports for feature services.
- `store/`: Shared global UI state only.
- `hooks/`: Generic reusable React hooks.
- `i18n/`: Internationalization configuration and modules.
- `themes/`: Design tokens and global styles.

## Mock BFF Architecture
Default development calls use `NEXT_PUBLIC_API_URL=/api`, so feature services go through Next.js route handlers under `src/app/api/**`. These mock BFF handlers emit the same HTTP contract shape as the real API and are disabled in production runtime unless explicitly opted in.

**Flow**: `Browser -> src/http/request.ts -> /api/* -> mock BFF route handlers`

## Authentication Flow
The scaffold uses httpOnly mock session cookies for local auth. Protected routes are managed by `AuthenticatedProviders` plus `AuthGuard` in the `(protected)` layout.

```typescript
// app/(protected)/layout.tsx
import { AuthGuard } from '@/features/auth';
import { AuthenticatedProviders } from '@/providers/authenticated-providers';

export default function ProtectedLayout({ children }) {
  return (
    <AuthenticatedProviders>
      <AuthGuard>{children}</AuthGuard>
    </AuthenticatedProviders>
  );
}
```

## Route Groups
- `(auth)`: Public auth pages (login, register).
- `(protected)`: Authenticated route group enforced by middleware and `AuthGuard`.
- `(protected)/(console)`: Replaceable scaffold console pages.
- `(protected)/(devtools)`: Internal playground and demo routes.
- `(site)`: Public marketing and information pages.

> [!NOTE]
> **Production Ready**: Downstream apps should point `NEXT_PUBLIC_API_URL` at the real API or a same-origin proxy and keep the mock BFF disabled unless running a demo-only deployment.

## Related Skills

- [`grill-before-build`](../../../../.agents/skills/grill-before-build/): Interview before any strategy-level decision lands as code.
- [`pr-description-writer`](../../../../.agents/skills/pr-description-writer/): The PR body is where a strategic decision first crystallizes.
- [`web-design-guidelines`](../web-design-guidelines/): Strategy shapes design language.
