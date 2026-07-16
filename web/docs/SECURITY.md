# Web Browser Security Boundary

This document defines the browser-facing response policy owned by the Luas Web scaffold. It keeps
application defaults, deployment controls, and downstream product policy separate so a header is
not mistaken for a complete security architecture.

## Ownership

| Owner | Responsibility |
|---|---|
| Luas Web runtime | Emit the reviewed response-header floor from `security-headers.ts`, route it through `next.config.ts`, and keep mock-session navigation checks in `src/proxy.ts`. |
| Route or starter | Own authentication, authorization, private cache headers, cookies, CORS where applicable, and any provider-specific browser connection policy. |
| Downstream deployment | Terminate trusted TLS, redirect HTTP to HTTPS, configure WAF/CDN behavior, avoid duplicate response headers, and decide whether subdomains and preload belong to its HSTS policy. |
| Downstream product | Extend CSP fetch directives, framing, and browser capabilities only for reviewed integrations such as analytics, direct asset transfer, identity, payments, maps, or embeds. |

`next.config.ts` applies the response policy to every path before filesystem and route resolution.
Cache behavior remains route-owned; do not add a global `Cache-Control` value to this policy.

## Default Policy

| Header | Default | Boundary |
|---|---|---|
| `Content-Security-Policy` | `base-uri 'self'; form-action 'self'; frame-ancestors 'none'; object-src 'none'` | Structural navigation and embedding floor. It is deliberately not presented as a complete script-injection policy. |
| `Permissions-Policy` | Disables browsing topics, camera, geolocation, microphone, payment, and USB | A downstream feature must explicitly review and enable a capability before using it. |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Keeps same-origin diagnostics while reducing cross-origin URL disclosure. |
| `Strict-Transport-Security` | `max-age=31536000` in production builds only | Covers the current host after a trusted HTTPS response. Luas does not claim ownership of subdomains or browser preload registration. |
| `X-Content-Type-Options` | `nosniff` | Prevents MIME-type guessing. |
| `X-DNS-Prefetch-Control` | `on` | Preserves the existing navigation-performance choice; it is not a security guarantee. |
| `X-Frame-Options` | `DENY` | Compatibility defense for clients that do not enforce CSP `frame-ancestors`. |
| `X-Permitted-Cross-Domain-Policies` | `none` | Rejects legacy cross-domain policy files. |
| `X-XSS-Protection` | `0` | Disables legacy browser XSS filters that can create unsafe transformations. Modern protection belongs in CSP and safe rendering. |

The structural CSP intentionally omits `default-src`, `script-src`, `style-src`, `connect-src`, and
other fetch directives. Adding a per-request nonce in Next.js requires dynamic rendering, while a
static broad policy can silently break direct asset grants, external identity, analytics, images,
fonts, or downstream APIs. Luas therefore does not add `'unsafe-inline'` and call the result a strict
CSP. A downstream app that owns its complete origin catalog should add a nonce- or hash-based policy
and remeasure rendering, caching, and route bundles. See the official
[Next.js CSP guide](https://nextjs.org/docs/app/guides/content-security-policy) and
[OWASP CSP guidance](https://cheatsheetseries.owasp.org/cheatsheets/Content_Security_Policy_Cheat_Sheet.html).

## Next.js Proxy

Next.js 16 renamed the deprecated `middleware.ts` convention to `proxy.ts`. Because this scaffold's
App Router is under `src/app`, Luas places the convention at `src/proxy.ts` and uses it only for
early navigation checks in `mock-session` mode:

- it verifies the signed mock cookie before protected render work;
- it redirects unauthenticated mock users to login and authenticated mock users away from public-only auth pages;
- it passes through `api-session` and `client-session` modes;
- it does not replace authentication or authorization in Route Handlers, Server Actions, or the Go API.

Keep the matcher narrow. Proxy is a network interception boundary, not a general application
middleware registry.

## Downstream Changes

When a product needs to be embedded, change both `frame-ancestors` and `X-Frame-Options`; do not
weaken only one and leave contradictory policy. When it needs a disabled browser capability, narrow
the `Permissions-Policy` allowlist to the exact origin. When it introduces a full CSP, inventory all
browser fetch destinations and test production navigation, RSC requests, optional analytics, API
adapters, and direct asset transfers.

HSTS is honored only after HTTPS. The deployment must own a valid certificate and HTTP-to-HTTPS
redirect. Add `includeSubDomains` or `preload` only after every affected hostname is permanently
HTTPS-capable and the organization accepts the recovery constraints. Configure one header owner at
the application or ingress rather than emitting conflicting duplicates.

## Verification

```bash
(cd web && pnpm vitest run src/test/security-headers.test.ts src/test/auth-runtime-boundary.test.ts)
(cd web && pnpm build)
(cd web && pnpm proxy:check)
bash web/scripts/verify-container.sh luas-web:container-check
python3 .agents/skills/luas-framework-review/scripts/check-web-security-boundary.py
make governance
```

The container verifier starts the real standalone image, requests `/login`, and requires one exact
value for every production security header. It then starts the same image with the explicit
production mock opt-in and proves an unauthenticated `/console?tab=security` request receives the
exact `307` login redirect. Every production build also parses
`.next/server/functions-config-manifest.json` and requires the Node.js Proxy plus all five exact
matchers, so a misplaced or unassembled convention cannot produce a false-green build. Static tests
reject header injection, duplicate names, the deprecated `middleware.ts` convention, and accidental
broad or unsafe CSP directives.

On the local Next.js 16.2.9 production build, `/login` moved from four to nine security-policy
headers. The HTTP/1.1 response-header block increased from 487 to 795 bytes (+308 bytes), while the
HTML body remained 41,866 bytes and every reviewed first-load JavaScript route measurement remained
unchanged. This is local response/build evidence, not a network-wide transfer guarantee; HTTP/2 and
HTTP/3 header compression, intermediaries, and downstream policy affect wire cost.

The final local arm64 image measured 65,944,285 bytes versus 65,949,553 bytes at the preceding
container-supply-chain baseline (-5,268 bytes, -0.008%, effectively flat). Its Trivy 0.72.0
CycloneDX inventory remains 39 components with zero recorded vulnerabilities, and the live gate
reported zero vulnerabilities and zero secrets. Treat the tiny image delta as build noise, not a
performance claim.
