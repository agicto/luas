#!/usr/bin/env python3

"""Keep the Web browser response and Next.js Proxy boundaries aligned."""

from __future__ import annotations

import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]


def read(relative_path: str) -> str:
    return (ROOT / relative_path).read_text(encoding="utf-8")


def require_all(
    failures: list[str], relative_path: str, markers: tuple[str, ...]
) -> None:
    path = ROOT / relative_path
    if not path.exists():
        failures.append(f"{relative_path} is missing")
        return

    content = read(relative_path)
    for marker in markers:
        if marker not in content:
            failures.append(f"{relative_path} must contain {marker!r}")


def require_absent(
    failures: list[str], relative_path: str, markers: tuple[str, ...]
) -> None:
    path = ROOT / relative_path
    if not path.exists():
        failures.append(f"{relative_path} is missing")
        return

    content = read(relative_path)
    for marker in markers:
        if marker in content:
            failures.append(f"{relative_path} must not contain {marker!r}")


def require_missing(failures: list[str], relative_path: str) -> None:
    if (ROOT / relative_path).exists():
        failures.append(f"{relative_path} must remain removed")


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "web/security-headers.ts",
        (
            "export function getBrowserSecurityHeaders",
            "key: 'Content-Security-Policy'",
            "key: 'Permissions-Policy'",
            "key: 'Referrer-Policy'",
            "key: 'Strict-Transport-Security'",
            "key: 'X-Content-Type-Options'",
            "key: 'X-DNS-Prefetch-Control'",
            "key: 'X-Frame-Options'",
            "key: 'X-Permitted-Cross-Domain-Policies'",
            "key: 'X-XSS-Protection'",
            "value: 'max-age=31536000'",
            "value: 'DENY'",
            "value: 'nosniff'",
            "value: '0'",
            "\"base-uri 'self'\"",
            "\"form-action 'self'\"",
            "\"frame-ancestors 'none'\"",
            "\"object-src 'none'\"",
            "'browsing-topics=()'",
            "'camera=()'",
            "'geolocation=()'",
            "'microphone=()'",
            "'payment=()'",
            "'usb=()'",
            "options.production",
        ),
    )
    require_absent(
        failures,
        "web/security-headers.ts",
        (
            "'unsafe-inline'",
            "'unsafe-eval'",
            "includeSubDomains",
            "preload",
            "SAMEORIGIN",
        ),
    )
    require_all(
        failures,
        "web/next.config.ts",
        (
            "from './security-headers'",
            "headers: getBrowserSecurityHeaders({",
            "production: process.env.NODE_ENV === 'production'",
            "source: '/:path*'",
        ),
    )
    require_absent(
        failures,
        "web/next.config.ts",
        (
            "X-Frame-Options",
            "Content-Security-Policy",
            "Strict-Transport-Security",
        ),
    )

    require_missing(failures, "web/middleware.ts")
    require_missing(failures, "web/proxy.ts")
    require_all(
        failures,
        "web/src/proxy.ts",
        (
            "export async function proxy(request: NextRequest)",
            "getAuthRuntimeMode() !== 'mock-session'",
            "getMockSessionCookieName()",
            "verifySession(raw)",
            "'/console/:path*'",
            "'/login'",
            "'/register'",
        ),
    )
    require_absent(
        failures,
        "web/src/proxy.ts",
        (
            "export async function middleware(",
            "Authorization",
            "fetch(",
        ),
    )
    require_all(
        failures,
        "web/scripts/check-proxy-manifest.mjs",
        (
            ".next/server/functions-config-manifest.json",
            "manifest?.functions?.['/_middleware']",
            "runtime !== 'nodejs'",
            "expectedMatchers",
            "matcher?.originalSource",
            "new RegExp(matcher.regexp)",
            "Proxy manifest check passed",
        ),
    )
    require_all(
        failures,
        "web/package.json",
        (
            '"proxy:check": "node ./scripts/check-proxy-manifest.mjs"',
            "next build && node ./scripts/check-proxy-manifest.mjs && node ./scripts/check-route-bundle-budget.mjs",
        ),
    )

    require_all(
        failures,
        "web/src/test/security-headers.test.ts",
        (
            "emits unique, injection-safe common headers",
            "uses a structural CSP without weakening script execution policy",
            "adds host-only HSTS in production",
            "uses the Next 16 proxy convention",
            "expect(headers).toHaveLength(8)",
        ),
    )
    require_all(
        failures,
        "web/src/test/auth-runtime-boundary.test.ts",
        (
            "makes proxy enforcement conditional",
            "source('proxy.ts')",
            "export async function proxy(",
        ),
    )
    require_all(
        failures,
        "web/scripts/verify-container.sh",
        (
            "\"content-security-policy\": \"base-uri 'self'; form-action 'self'; frame-ancestors 'none'; object-src 'none'\"",
            '"strict-transport-security": "max-age=31536000"',
            '"x-frame-options": "DENY"',
            '"permissions-policy": "browsing-topics=(), camera=(), geolocation=(), microphone=(), payment=(), usb=()"',
            "validated {len(expected)} browser security response headers",
            "browser security headers: 9 verified",
            "--env MOCK_BFF_ENABLED=true",
            "mock-session Proxy returned HTTP",
            "expected_proxy_location='/login?returnUrl=%2Fconsole%3Ftab%3Dsecurity'",
            "mock-session Proxy: %s -> %s",
        ),
    )

    require_all(
        failures,
        "web/docs/SECURITY.md",
        (
            "# Web Browser Security Boundary",
            "## Ownership",
            "## Default Policy",
            "structural CSP",
            "not presented as a complete script-injection policy",
            "production builds only",
            "includeSubDomains",
            "preload",
            "## Next.js Proxy",
            "functions-config-manifest.json",
            "not replace authentication or authorization",
            "## Downstream Changes",
            "scripts/verify-container.sh",
        ),
    )
    require_all(
        failures,
        "web/README.md",
        ("docs/SECURITY.md", "`src/proxy.ts` and `AuthGuard`"),
    )
    require_all(
        failures,
        "web/AGENTS.md",
        (
            "#### Browser Security Response Rules",
            "docs/SECURITY.md",
            "Next.js 16 `proxy.ts` convention",
        ),
    )
    require_all(
        failures,
        "AGENTS.md",
        ("web/docs/SECURITY.md", "check-web-security-boundary.py"),
    )
    require_all(
        failures,
        ".agents/skills/luas-framework-review/SKILL.md",
        ("scripts/check-web-security-boundary.py",),
    )
    require_all(
        failures,
        "docs/FRAMEWORK_QUALITY_ROADMAP.md",
        (
            "Completed P1 — Web Browser Security Response Boundary",
            "Next.js 16 `proxy.ts`",
            "nine exact production response headers",
        ),
    )
    require_all(
        failures,
        "Makefile",
        ("check-web-security-boundary.py",),
    )

    if failures:
        print("Web security boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"- {failure}", file=sys.stderr)
        return 1

    print("Web security boundary check passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
