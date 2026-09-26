# Admin Console Security

## Static Trust Boundary

Every byte under `dist/` is public. Every `VITE_*` value is visible to users
and must be treated as browser configuration, never a secret. The static host
cannot read private environment variables, set HttpOnly cookies from
application code, or enforce authorization.

## Authentication

The preferred protected-app flow is:

```text
Admin browser application
  -> same-origin /api operation
  -> built-in Go browser-session adapter or reviewed gateway
  -> HttpOnly session cookie / fixed upstream mapping
  -> Luas API authorization
```

The gateway owns:

- `Secure`, HttpOnly, appropriately scoped cookies;
- exact Origin checks and CSRF policy for unsafe requests;
- fixed allowlisted upstream operations;
- credential-to-Bearer mapping when the Go API still expects an opaque token;
- timeout and response-size limits;
- safe forwarding and public error normalization.

The built-in adapter exposes `/v1/browser/auth/login`, `/me`, and `/logout`
when `BROWSER_SESSION_ENABLED=true`; it is fail-closed with deterministic `503`
responses when disabled. Production requires an exact HTTPS
`BROWSER_SESSION_ORIGIN` and PostgreSQL persistence.

The existing Go `/v1/login` bearer response is server-to-server/API behavior.
Do not put its `access_token` in `localStorage`, `sessionStorage`, IndexedDB,
Zustand persistence, query cache, URLs, logs, or analytics. The adapter proves
identity but does not invent platform authority. Protected Admin remains
incomplete until its login/session UI and a separate system-operator
authorization boundary are assembled.

## Cross-Origin APIs

Same-origin CDN routing is preferred. If a separate API origin is unavoidable:

- allowlist exact production origins;
- use HTTPS;
- configure credentials deliberately;
- never combine credentialed requests with wildcard CORS;
- define CSRF protection independently of CORS;
- expose only required safe response headers;
- keep preflight caching bounded and reviewed.

## Client Controls

- `src/http/client.ts` enforces fixed relative paths, credentials mode,
  timeout, JSON parsing, response-size bounds, envelope shape, and stable
  errors.
- Feature services validate important successful payloads with Zod.
- TanStack Query never persists by default.
- Mutations do not retry automatically.
- `request_id` may be displayed for support; provider errors and private
  payloads may not.

## CDN Controls

Configure CSP, MIME sniffing protection, framing denial, referrer policy,
permissions policy, TLS, and HSTS at the CDN. Keep `index.html` revalidated and
hashed assets immutable. Disable public source maps unless an authenticated
private symbol pipeline owns them.

A client-side route guard, hidden navigation item, or minified bundle is not a
security boundary. Every protected API operation revalidates identity,
authorization, tenant context, and resource ownership server-side.
