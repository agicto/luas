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
API_ADAPTER_ENABLED=false
NEXT_PUBLIC_OPTIONAL_FEATURES=
MOCK_BFF_ENABLED=false
```

Most `/api/*` route behavior is the development mock BFF. Auth and enabled optional feature routes
can select the shipped production adapter. Mock behavior is available outside production by default, while
production returns `503 COMMON.SERVICE_UNAVAILABLE` unless a production backend or explicit
demo-only `MOCK_BFF_ENABLED=true` is configured.

The Go JWT auth endpoints and Web browser auth endpoints are not drop-in compatible. Luas ships a
same-origin server adapter that performs the mapping without exposing bearer tokens to browser
JavaScript. Changing `NEXT_PUBLIC_API_URL` alone still does not enable it; read
[../contracts/AUTHENTICATION.md](../contracts/AUTHENTICATION.md) for its server-only configuration,
cookie, timeout, trusted-proxy, and stateless logout contract.

See [docs/MOCK_BFF.md](docs/MOCK_BFF.md) before replacing, deleting, or intentionally enabling the mock BFF.

When intentionally enabling the mock BFF in production, set a strong `SESSION_SECRET` because the scaffold mock auth session cookie is HMAC-signed:

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
│   └── api/                # Browser HTTP routes: mock behavior + fixed production adapters
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

The Web browser auth contract is implemented under `src/app/api/auth/`:

- `POST /api/auth/login`
- `POST /api/auth/register`
- `GET /api/auth/me`
- `POST /api/auth/logout`

Local development uses the mock BFF by default. Demo account:

```text
admin@example.com / admin123
```

The login form receives this preset only while the Web owns a mock session. Normal production and
real-API modes stay blank; an explicitly enabled demo deployment remains usable and shows the
preset without placing it in client static chunks. Mock session cookies use the `__Host-` prefix
and secure attributes in production, and unsafe mock BFF routes require an exact same-origin
browser request.

`middleware.ts` and `AuthGuard` provide protected-route navigation UX. Go endpoints remain the
authorization boundary for production operations.

For the Luas Go API, enable the shipped production adapter while keeping browser requests on `/api`:

```env
API_ADAPTER_ENABLED=true
API_UPSTREAM_URL=http://api:8025/v1
API_UPSTREAM_TIMEOUT_MS=5000
API_UPSTREAM_MAX_RESPONSE_BYTES=1048576
API_CLIENT_IP_HEADER=x-real-ip
```

The adapter takes precedence over mock auth, stores the API JWT in an HttpOnly host cookie, resolves
protected sessions on the server, and leaves unrelated mock routes disabled. Downstream apps using
another identity provider can keep `client-session` mode and replace this adapter seam.

## Organization Feature

Enable the browser feature only when the API process also enables the optional organization starter:

```env
NEXT_PUBLIC_OPTIONAL_FEATURES=organization
```

The first browser workflow provides the organization directory, creation, URL-scoped switching,
active-context verification, and basic rename settings. Development uses a replaceable in-memory
mock; production uses fixed same-origin adapter routes and the HttpOnly API session. Selection is
derived from `/console/organizations/:id` and is never persisted globally. See
[docs/ORGANIZATIONS.md](docs/ORGANIZATIONS.md) and
[../contracts/ORGANIZATIONS.md](../contracts/ORGANIZATIONS.md).

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
