# Admin Console Security

## Static Trust Boundary

Every byte under `dist/` is public. Every `VITE_*` value is visible to users
and must be treated as browser configuration, never a secret. The static host
cannot read private environment variables, set HttpOnly cookies from
application code, or enforce authorization.

## Authentication

The preferred protected-app flow is:

```text
Browser SPA
  -> same-origin /api operation
  -> reviewed browser gateway or Go adapter
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

The existing Go `/v1/login` bearer response is server-to-server/API behavior.
Do not put its `access_token` in `localStorage`, `sessionStorage`, IndexedDB,
Zustand persistence, query cache, URLs, logs, or analytics.

## Platform Operator Sign-In

The optional `operator` API starter is the shipped Go browser adapter for this
console ([`../../contracts/OPERATORS.md`](../../contracts/OPERATORS.md)). Enable it
in the console build with `VITE_OPTIONAL_FEATURES=operator`; without the flag the
console has no sign-in and no protected feature:

- `POST /v1/operator/session` sets an HttpOnly `SameSite=Strict` cookie; the
  credential never reaches JavaScript.
- The session response carries a CSRF token that `src/http/client.ts` keeps in
  memory only and sends as `X-CSRF-Token` on unsafe requests. After
  `OPERATOR.CSRF_REJECTED` the client refetches the session once and retries.
- `/console` routes require a current operator session; `401`,
  `OPERATOR.FORBIDDEN`, and `AUTH.ACCOUNT_DISABLED` return to `/login`. This
  guard is UX only; the API authorizes every operator request.
- The console origin must be listed in `OPERATOR_ALLOWED_ORIGINS`, and `/api/*`
  must reach the Go API on the same origin so the cookie stays first-party.
- Operator access is granted only with `luas operator:grant <email>`.
- `organization`, `webhook`, and `notification` in `VITE_OPTIONAL_FEATURES` add operator screens
  for those API starters. They are support views, not tenant self-service: an operator reads any
  organization's members, webhook endpoints, and deliveries, and the notification delivery ledger,
  without becoming an organization member. The API never returns signing secrets, event payloads,
  notification content, or recipient addresses on these routes, so the console cannot show them.
  The only write is webhook delivery replay, which is audited with the operator as actor.

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
