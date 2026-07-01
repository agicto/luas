---
name: environment-config
description: Strict rules for environment variable management using Zod validation and src/config/env.ts.
---

# environment-config

## Overview

This skill provides guidelines for managing environment variables in the project. It ensures type safety and prevents runtime errors by using a single source of truth and schema validation.

## Guidelines

### 1. Single Source of Truth
All environment variables MUST be defined, validated, and exported from `src/config/env.ts`.

**❌ DO NOT:**
```typescript
const apiUrl = process.env.NEXT_PUBLIC_API_URL; // Unsafe, untyped
```

**✅ DO:**
```typescript
import { env } from '@/config/env';
const apiUrl = env.NEXT_PUBLIC_API_URL; // Typed, validated
```

### 2. Validation (Zod)
The project uses `zod` to validate environment variables at runtime. If a required variable is missing, the application will fail to start with a clear error message.

### 3. Adding a New Environment Variable
1. **Add to `.env.local`**: Define the variable and its value.
2. **Update `src/config/env.ts`**:
   - Add the validation schema to the `envSchema` object.
   - Ensure it's correctly mapped in the `env` export.

### 4. Supported Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXT_PUBLIC_API_URL` | No | `/api` | Base URL for API requests. |
| `NEXT_PUBLIC_APP_URL` | No | `http://localhost:3000` | Absolute site URL for metadata, sitemap, and robots. |
| `NEXT_PUBLIC_DEFAULT_LOCALE` | No | `zh-Hans` | Default locale. Must be one of `src/i18n/locales.ts`. |
| `NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED` | No | `true` | Shows or hides the language switcher. |
| `NEXT_PUBLIC_GA_MEASUREMENT_ID` | No | - | Google Analytics ID. |
| `NODE_ENV` | No | `development` | App environment (`development` \| `production` \| `test`). |
| `MOCK_BFF_ENABLED` | Production opt-in only | `false` | Enables development mock BFF route handlers in production runtime. |
| `SESSION_SECRET` | Production runtime | - | Server-only secret for HMAC-signed mock auth cookies. |

> [!IMPORTANT]
> Always use the `env` object from `@/config/env` to access environment variables. Directly accessing `process.env` is strictly prohibited outside of the config file.

`src/test/env-contract.test.ts` enforces this rule for production source files.

## Related Skills

- [`utility-tooling`](../utility-tooling/): Config helpers and utilities live alongside env handling.
- [`api-error-handling`](../api-error-handling/): Config errors should surface as proper error UX.
- [`verification-before-completion`](../../../../.agents/skills/verification-before-completion/): Verify env-dependent paths in dev before claiming done.
