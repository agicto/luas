---
name: environment-config
description: Change validated Web environment configuration. Use for public/server-only variables, runtime versus build-time values, defaults, or secret boundaries.
---

# environment-config

## Overview

This skill provides guidelines for managing environment variables in the project. It ensures type safety without leaking server-only names, validation, or secrets into browser bundles.

## Guidelines

### 1. Explicit Runtime Entries

- Browser-safe values are resolved without a schema-library runtime in `src/config/env.ts` and imported from `@/config/env`.
- Server-only values belong in `src/config/server-env.ts` and are imported from `@/config/server-env` only by server modules.
- Authoritative Zod schemas and shared preprocessors belong in the server-only `src/config/env-validation.ts`; root layout loads `server-env.ts` so public and server values fail fast during build/startup.
- Never export `server-env` from `src/config/index.ts` or another client-reachable barrel.

**❌ DO NOT:**
```typescript
const apiUrl = process.env.NEXT_PUBLIC_API_URL; // Unsafe, untyped
```

**✅ DO:**
```typescript
import { env } from '@/config/env';
const apiUrl = env.NEXT_PUBLIC_API_URL; // Typed, validated
```

```typescript
import { serverEnv } from '@/config/server-env';
const mockEnabled = serverEnv.MOCK_BFF_ENABLED; // Server modules only
```

### 2. Validation (Zod)
The project uses `zod` to validate environment variables at runtime. If a required variable is missing, the application will fail to start with a clear error message.

### 3. Adding a New Environment Variable
1. **Add to `.env.local`**: Define the variable and its value.
2. **Choose the runtime boundary**:
   - Add `NEXT_PUBLIC_*` and browser-safe build values to `src/config/env.ts`.
   - Add secrets and server-only runtime values to `src/config/server-env.ts`.
3. **Update the matching Zod schema**:
   - Add browser-safe values to the lightweight `env` resolver and `publicEnvSchema`.
   - Add server-only values to `serverEnvSchema` and the `serverEnv` parse input.

### 4. Supported Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXT_PUBLIC_API_URL` | No | `/api` | Base URL for API requests. |
| `NEXT_PUBLIC_APP_URL` | No | `http://localhost:3000` | Absolute site URL for metadata, sitemap, and robots. |
| `NEXT_PUBLIC_DEFAULT_LOCALE` | No | `zh-Hans` | Default locale. Must be one of `src/i18n/locales.ts`. |
| `NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED` | No | `true` | Shows or hides the language switcher. |
| `NEXT_PUBLIC_GA_MEASUREMENT_ID` | No | - | Google Analytics ID. |
| `NODE_ENV` | No | `development` | App environment (`development` \| `production` \| `test`). |
| `AUTH_ADAPTER_ENABLED` | No | `false` | Enables the same-origin server adapter for the Luas Go user starter. |
| `AUTH_API_URL` | Auth adapter runtime | - | Server-only Go API base URL including its `/v1` prefix. |
| `AUTH_API_TIMEOUT_MS` | No | `5000` | Server-to-server auth timeout, from 100 to 30000 milliseconds. |
| `AUTH_CLIENT_IP_HEADER` | Production auth adapter | - | Ingress-overwritten source header containing one client IP, such as `X-Real-IP`; appended chains are rejected. |
| `MOCK_BFF_ENABLED` | Production opt-in only | `false` | Enables development mock BFF route handlers in production runtime. |
| `SESSION_SECRET` | Production mock BFF | - | Server-only secret for HMAC-signed mock auth cookies; required when `MOCK_BFF_ENABLED=true`. |

> [!IMPORTANT]
> Use `env` for browser-safe values and `serverEnv` for server-only values. Direct `process.env` access is restricted to the two environment entry modules and test setup.

`src/test/env-contract.test.ts` enforces this rule, keeps server values out of the client entry and
config barrel, verifies the production adapter's same-origin/upstream/client-IP combination, and
requires a strong `SESSION_SECRET` only for production mock BFF opt-in.

## Related Skills

Select another skill only when its distinct concern is active.

- [`utility-tooling`](../utility-tooling/): Config helpers and utilities live alongside env handling.
- [`api-error-handling`](../api-error-handling/): Config errors should surface as proper error UX.
