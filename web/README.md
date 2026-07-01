# Luas Web

Luas Web is the Next.js half of the Luas scaffold. It provides a feature-first React app, development mock BFF endpoints, protected console routes, i18n, typed environment config, and a small HTTP client for talking to the API half.

## Stack

| Area | Tooling |
|---|---|
| Framework | Next.js 16.2.9 App Router |
| UI | React 19.2.7, Tailwind CSS 4, Radix UI, lucide-react |
| State | TanStack Query 5, Zustand 5 |
| i18n | next-intl 4 |
| Tests | Vitest 4, Testing Library, happy-dom |
| Tooling | TypeScript 5.9, ESLint 9, Prettier |

## Quick Start

```bash
pnpm install
cp .env.example .env.local
pnpm dev
```

The app runs at [http://localhost:3000](http://localhost:3000).

Default development env:

```env
NEXT_PUBLIC_API_URL=/api
NEXT_PUBLIC_APP_URL=http://localhost:3000
NEXT_PUBLIC_DEFAULT_LOCALE=zh-Hans
NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED=true
MOCK_BFF_ENABLED=false
```

`/api/*` route handlers are the development mock BFF. They are available outside production by default, but production runtime returns `503 COMMON.SERVICE_UNAVAILABLE` unless `MOCK_BFF_ENABLED=true` is set explicitly. Downstream production apps should normally point `NEXT_PUBLIC_API_URL` at the real Luas API instead of enabling mock routes.

See [docs/MOCK_BFF.md](docs/MOCK_BFF.md) before replacing, deleting, or intentionally enabling the mock BFF.

For production runtime, set a strong `SESSION_SECRET` because the scaffold mock auth session cookie is HMAC-signed:

```bash
openssl rand -hex 32
```

## Project Structure

```text
src/
├── app/                    # Next.js App Router
│   ├── (auth)/             # Public auth routes
│   ├── (protected)/        # Authenticated route groups
│   ├── (site)/             # Public site pages
│   └── api/                # Mock BFF route handlers
├── components/             # Shared UI and layout components
├── features/               # Feature-first folders
├── http/                   # Axios wrapper and error normalization
├── i18n/                   # next-intl modules and helpers
├── providers/              # App providers
├── store/                  # Shared global state
├── test/                   # Test setup
├── themes/                 # Design tokens
└── utils/                  # Pure utilities
```

## Auth

The web scaffold includes mock BFF auth endpoints under `src/app/api/auth/`:

- `POST /api/auth/login`
- `POST /api/auth/register`
- `GET /api/auth/me`
- `POST /api/auth/logout`

Demo account:

```text
admin@example.com / admin123
```

Protected routes are enforced by `middleware.ts` and `AuthGuard`.

Before shipping a downstream app, replace these mock auth routes with the real API-backed auth flow or keep them disabled in production.

## HTTP Contract

The default request client is configured by `NEXT_PUBLIC_API_URL`. Error handling understands the Go API error shape:

```json
{
  "code": 404,
  "error_code": "COMMON.NOT_FOUND",
  "message": "Not found",
  "request_id": "req_123"
}
```

See [../contracts/README.md](../contracts/README.md) for the shared contract.

## Scripts

```bash
pnpm dev
pnpm type-check
pnpm lint
pnpm test -- --run
pnpm test:coverage
pnpm build
```

Use `make check` from the repository root to run the canonical API and Web verification tiers together.
