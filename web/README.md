# Luas Web

Luas Web is the Next.js half of the Luas scaffold. It provides a feature-first React app, development mock BFF endpoints, protected console routes, i18n, typed environment config, and a small HTTP client for talking to the API half.

## Stack

| Area      | Tooling                                              |
| --------- | ---------------------------------------------------- |
| Framework | Next.js 16.2.9 App Router                            |
| UI        | React 19.2.7, Tailwind CSS 4, Radix UI, lucide-react |
| State     | TanStack Query 5, Zustand 5                          |
| i18n      | next-intl 4                                          |
| Tests     | Vitest 4, Testing Library, happy-dom                 |
| Tooling   | TypeScript 5.9, ESLint 9, Prettier                   |

## Quick Start

```bash
corepack pnpm install
cp .env.example .env.local
corepack pnpm dev
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

The Go authentication-session endpoints and Web browser auth endpoints are not drop-in compatible. Luas ships a
same-origin server adapter that performs the mapping without exposing bearer tokens to browser
JavaScript. Changing `NEXT_PUBLIC_API_URL` alone still does not enable it; read
[../contracts/AUTHENTICATION.md](../contracts/AUTHENTICATION.md) for its server-only configuration,
cookie, timeout, trusted-proxy, and remote-revocation contract.

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

## Internationalization

Web uses `next-intl` with `en-US` as the canonical message schema and
`zh-Hans` as a fully checked translation locale. Request resolution follows a
valid locale cookie, weighted `Accept-Language`, then the configured default.
Client routes receive only their owned message namespaces to avoid shipping the
entire catalog.

Read [src/i18n/README.md](src/i18n/README.md) before adding a locale,
namespace, translated error, or client-side message dependency.

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

`src/proxy.ts` and `AuthGuard` provide protected-route navigation UX. Go endpoints remain the
authorization boundary for production operations.

For the Luas Go API, enable the shipped production adapter while keeping browser requests on `/api`:

```env
API_ADAPTER_ENABLED=true
API_UPSTREAM_URL=http://api:8025/v1
API_UPSTREAM_TIMEOUT_MS=5000
API_UPSTREAM_MAX_RESPONSE_BYTES=1048576
API_CLIENT_IP_HEADER=x-real-ip
```

The adapter takes precedence over mock auth, stores the opaque API credential in an HttpOnly host
cookie, resolves protected sessions on the server, remotely revokes logout, and leaves unrelated mock routes disabled. Downstream apps using
another identity provider can keep `client-session` mode and replace this adapter seam.

## Organization Feature

Enable the browser feature only when the API process also enables the optional organization starter:

```env
NEXT_PUBLIC_OPTIONAL_FEATURES=organization
```

The browser workflow provides organization directory/create, URL-scoped switching, active-context
verification, profile, member, invitation, and ownership management. Development uses a replaceable
in-memory mock; production uses fixed same-origin adapter routes and the HttpOnly API session.
Selection is derived from `/console/organizations/:id` and is never persisted globally. See
[docs/ORGANIZATIONS.md](docs/ORGANIZATIONS.md) and
[../contracts/ORGANIZATIONS.md](../contracts/ORGANIZATIONS.md).

## Permission Feature

Enable permission only with its organization dependency in both deployable halves:

```env
OPTIONAL_STARTERS=organization,permission
NEXT_PUBLIC_OPTIONAL_FEATURES=organization,permission
```

The organization detail page then lazy-loads access-role CRUD and member assignment management.
Every browser call carries the URL-derived organization ID; strict schemas validate responses before
caching, and Go remains authoritative. See [docs/PERMISSIONS.md](docs/PERMISSIONS.md) and
[../contracts/PERMISSIONS.md](../contracts/PERMISSIONS.md).

## Notification Feature

Notification is independent of organization and must be selected in both deployable halves:

```env
OPTIONAL_STARTERS=notification
NEXT_PUBLIC_OPTIONAL_FEATURES=notification
```

The console then loads a user-scoped notification center with unread status, race-safe read state,
and explicit in-app/email preferences. Development may use the isolated mock BFF store; production
uses fixed same-origin adapter routes and the HttpOnly API session. Provider content is rendered as
plain text, action URLs are restricted to safe local paths, and preference mutations are same-origin
only. See [docs/NOTIFICATIONS.md](docs/NOTIFICATIONS.md) and
[../contracts/NOTIFICATIONS.md](../contracts/NOTIFICATIONS.md).

## Asset Feature

Asset is user-scoped and must be selected in both deployable halves:

```env
OPTIONAL_STARTERS=asset
NEXT_PUBLIC_OPTIONAL_FEATURES=asset
```

The console then exposes private upload, lifecycle status, download, and delete operations. Grants
remain ephemeral, successful payloads are strictly parsed, and browser transfers reject unsafe URLs,
headers, credentials, and redirects. Development uses a bounded per-user mock object store;
production uses fixed same-origin management adapters and direct short-lived R2 grants. See
[docs/ASSETS.md](docs/ASSETS.md) and [../contracts/ASSETS.md](../contracts/ASSETS.md).

## Setting Feature

Setting depends on organization and must be selected in both deployable halves:

```env
OPTIONAL_STARTERS=organization,setting
NEXT_PUBLIC_OPTIONAL_FEATURES=organization,setting
```

The console validates the finite typed catalog, preserves ETag/version preconditions, and exposes
only real user and organization preferences. See [docs/SETTINGS.md](docs/SETTINGS.md) and
[../contracts/SETTINGS.md](../contracts/SETTINGS.md).

## Usage Feature

Usage also depends on organization and is read-only in the browser:

```env
OPTIONAL_STARTERS=organization,usage
NEXT_PUBLIC_OPTIONAL_FEATURES=organization,usage
```

The console validates the complete finite metric catalog and displays private current-period user
or owner/admin organization summaries. Event recording, atomic consumption, quota writes, receipts,
and billing semantics remain server-side. See [docs/USAGE.md](docs/USAGE.md) and
[../contracts/USAGE.md](../contracts/USAGE.md).

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

## Browser Security

Production responses use a centralized, executable browser security policy with deny-by-default
framing, a structural CSP floor, disabled unused browser capabilities, MIME protection, referrer
minimization, and host-only HSTS. The policy deliberately leaves provider-specific CSP fetch
origins and TLS/subdomain ownership to the downstream product and deployment. See
[docs/SECURITY.md](docs/SECURITY.md) before adding embeds, external browser integrations, camera or
payment capabilities, a nonce/hash CSP, or ingress-owned security headers.

## Performance

Production builds enforce reviewed first-load JavaScript budgets from
[`performance-budgets.json`](performance-budgets.json) using Next.js route diagnostics. The raw
route value is the deterministic gate; gzip output is diagnostic. Synthetic Lighthouse evidence
and real-user Core Web Vitals remain separate measurement classes.

Read [docs/PERFORMANCE.md](docs/PERFORMANCE.md) before changing root providers, shared client
controls, analytics, large dependencies, or route-level lazy boundaries.

## Production Container

The production Dockerfile pins its frontend and Node base by digest, runs the same budgeted build,
and materializes a non-root standalone runtime without apk, Node headers, npm, Corepack, pnpm, or
Yarn. Build, start, health-check, and terminate it with:

```bash
bash scripts/verify-container.sh luas-web:container-check
```

The root [container security contract](../docs/CONTAINER_SECURITY.md) defines BuildKit evidence,
CycloneDX 1.7 inventory, Trivy policy, CI artifacts, and the downstream signing boundary.

## Scripts

```bash
corepack pnpm dev
corepack pnpm type-check
corepack pnpm lint
corepack pnpm test -- --run
corepack pnpm test:coverage
corepack pnpm build
corepack pnpm bundle:check
corepack pnpm bundle:analyze
bash scripts/verify-container.sh luas-web:container-check
```

Use `make check` from the repository root to run the canonical API, Next.js Web, and Admin Console
verification tiers together.
Dependency tooling is exact-versioned; use Corepack instead of an unrelated global pnpm. Root
vulnerability scanning, SBOM, update, and exception policy lives in
[`../docs/DEPENDENCY_SECURITY.md`](../docs/DEPENDENCY_SECURITY.md).
