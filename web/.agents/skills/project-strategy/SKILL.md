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
The scaffold uses httpOnly mock session cookies for local auth. `resolveAuthRuntimeMode()` selects
`mock-session` only when the mock BFF is available and the API target is same-origin `/api`;
external APIs and production proxies use `client-session`.

Protected routes resolve a serializable bootstrap on the server, then create one isolated Zustand
store per `AuthProvider`. A definitive mock session renders without a client `/auth/me` request;
client-owned real API sessions use one deduplicated request. Session absence (`401`), access denial
(`403`), and temporary resolution failure are separate states so infrastructure incidents do not
become false logout events.

```typescript
// app/(protected)/layout.tsx
import { AuthGuard } from '@/features/auth';
import { resolveAuthBootstrap } from '@/features/auth/server/bootstrap';
import { AuthenticatedProviders } from '@/providers/authenticated-providers';

export default async function ProtectedLayout({ children }) {
  const bootstrap = await resolveAuthBootstrap();
  return (
    <AuthenticatedProviders bootstrap={bootstrap}>
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
> **Production Ready**: Downstream apps should point `NEXT_PUBLIC_API_URL` at the real API or a same-origin proxy and keep the mock BFF disabled unless running a demo-only deployment. `AuthGuard` is UX; the API, Route Handlers, and Server Actions remain the authorization boundary.

## Related Skills

- [`grill-before-build`](../../../../.agents/skills/grill-before-build/): Interview before any strategy-level decision lands as code.
- [`pr-description-writer`](../../../../.agents/skills/pr-description-writer/): The PR body is where a strategic decision first crystallizes.
- [`web-design-guidelines`](../web-design-guidelines/): Strategy shapes design language.
