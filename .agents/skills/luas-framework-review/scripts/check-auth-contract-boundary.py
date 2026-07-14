#!/usr/bin/env python3

"""Keep the current Web browser auth and Go JWT boundary explicit."""

from __future__ import annotations

import re
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
            failures.append(f"{relative_path} must not contain stale marker {marker!r}")


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "CONTEXT.md",
        (
            "Production auth adapter",
            "browser auth contract vs API auth contract",
            "not interchangeable",
            "changing `NEXT_PUBLIC_API_URL` alone does not perform the mapping",
        ),
    )
    require_all(
        failures,
        "contracts/README.md",
        ("AUTHENTICATION.md", "explicit adapter"),
    )
    require_all(
        failures,
        "contracts/AUTHENTICATION.md",
        (
            "## Browser Auth Contract",
            "POST /api/auth/login",
            "GET /api/auth/me",
            "## Go API User Starter Contract",
            "POST /v1/login",
            "GET /v1/users/profile",
            "AUTH.INVALID_CREDENTIALS",
            "per-IP and normalized/hashed per-subject buckets",
            "SERVER_TRUSTED_PROXIES",
            "## Production Auth Adapter",
            "AUTH_ADAPTER_ENABLED=true",
            "AUTH_API_URL=http://api:8025/v1",
            "AUTH_CLIENT_IP_HEADER=x-real-ip",
            "__Host-luas_auth",
            "no refresh token, token denylist, or remote logout",
        ),
    )
    require_absent(
        failures,
        "contracts/AUTHENTICATION.md",
        ("does not yet ship a production adapter", "not ready-to-use"),
    )
    require_all(
        failures,
        "api/internal/modules/user/routes.go",
        (
            'r.POST("/login"',
            'r.POST("/register"',
            'auth.GET("/users/profile"',
            "protectPublicRoute",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/user/auth_abuse_guard.go",
        (
            "PerSubject",
            "crypto.SHA256Hex",
            "SuppressHeaders: true",
            "response.ErrorCodeRateLimited",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/user/service.go",
        ("dummyPasswordHash", "FindByLoginIdentifier", "domain.ErrInvalidCredentials"),
    )
    require_all(
        failures,
        "api/internal/bootstrap/http.go",
        ("configureTrustedProxies", "SetTrustedProxies"),
    )

    env_example = read("api/.env.example")
    for key in ("MIDDLEWARE_RATE_LIMIT_ENABLED", "AUTH_RATE_LIMIT_ENABLED"):
        if re.search(rf"^{key}=false(?:\s|$)", env_example, re.MULTILINE):
            failures.append(
                f"api/.env.example must not actively disable production-default {key}"
            )
    require_all(
        failures,
        "api/internal/modules/user/dto.go",
        ('json:"username"', 'json:"access_token"', 'json:"user"'),
    )
    require_all(
        failures,
        "web/src/features/auth/services/auth-service.ts",
        ("'/auth/login'", "'/auth/register'", "'/auth/me'", "'/auth/logout'"),
    )
    require_all(
        failures,
        "web/src/features/auth/types.ts",
        ("export interface AuthUser", "export interface LoginRequest", "email: string"),
    )
    auth_types = read("web/src/features/auth/types.ts")
    auth_user_match = re.search(
        r"export interface AuthUser\s*\{(?P<body>.*?)\}", auth_types, re.DOTALL
    )
    if not auth_user_match or re.search(r"\brole\s*:", auth_user_match.group("body")):
        failures.append(
            "web/src/features/auth/types.ts AuthUser must not invent a role before the "
            "permission starter exists"
        )
    require_all(
        failures,
        "web/src/config/server-env.ts",
        (
            "AUTH_ADAPTER_ENABLED",
            "AUTH_API_TIMEOUT_MS",
            "AUTH_API_URL",
            "AUTH_CLIENT_IP_HEADER",
            "NEXT_PUBLIC_API_URL must target the same-origin /api route",
        ),
    )
    require_all(
        failures,
        "web/src/config/auth-session.ts",
        (
            "__Host-luas_auth",
            "httpOnly: true",
            "sameSite: 'lax'",
            "secure: environment === 'production'",
        ),
    )
    require_all(
        failures,
        "web/src/app/api/_shared/mock-bff.ts",
        ("env.NEXT_PUBLIC_APP_URL", "parsedOrigin.origin !== allowedOrigin"),
    )
    require_all(
        failures,
        "web/src/features/auth/server/go-api-auth-adapter.ts",
        (
            "'users/profile'",
            "cache: 'no-store'",
            "redirect: 'error'",
            "headers.set('authorization'",
            "candidate.includes(',')",
            "generatedUsername",
            "mapGoApiUser",
        ),
    )
    require_all(
        failures,
        "web/src/features/auth/server/auth-adapter-route.ts",
        (
            "setApiSessionCookie",
            "getApiSessionToken",
            "clearApiSessionCookie",
            "resolveGoApiAuthBootstrap",
        ),
    )
    for route in ("login", "register", "me", "logout"):
        require_all(
            failures,
            f"web/src/app/api/auth/{route}/route.ts",
            ("resolveAuthRoute()",),
        )
    for test_file in (
        "web/src/test/auth-adapter-route.test.ts",
        "web/src/test/auth-route-backend.test.ts",
        "web/src/test/auth-session-cookie.test.ts",
        "web/src/test/go-api-auth-adapter.test.ts",
    ):
        if not (ROOT / test_file).exists():
            failures.append(f"{test_file} is missing")
    require_all(
        failures,
        "web/docs/AUTHENTICATION.md",
        (
            "../../contracts/AUTHENTICATION.md",
            "three resolution modes",
            "`api-session`",
            "`__Host-luas_auth`",
        ),
    )
    require_absent(
        failures,
        "web/docs/AUTHENTICATION.md",
        ("not a complete integration", "does not yet ship the production"),
    )
    require_all(
        failures,
        "web/src/i18n/modules/console/en-US.ts",
        ("contract-compatible production endpoint or adapter",),
    )
    require_all(
        failures,
        "web/src/i18n/modules/console/zh-Hans.ts",
        ("契约兼容的生产端点或适配器",),
    )

    roadmap = read("docs/STARTER_BUSINESS_ROADMAP.md")
    if not re.search(r"^\| `user` default starter .*\| Yes \|", roadmap, re.MULTILINE):
        failures.append(
            "docs/STARTER_BUSINESS_ROADMAP.md must mark the combined user starter ready only "
            "while the production auth adapter guardrails remain implemented"
        )

    if failures:
        print("Auth contract boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print("Auth contract boundary check passed (production adapter and auth ownership are explicit).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
