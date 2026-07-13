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


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "CONTEXT.md",
        ("browser auth contract vs API auth contract", "not interchangeable"),
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
            "does not yet ship a production adapter",
            "not ready-to-use",
        ),
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
    require_all(
        failures,
        "web/docs/AUTHENTICATION.md",
        ("../../contracts/AUTHENTICATION.md", "not a complete integration"),
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
    if not re.search(r"^\| `user` default starter .*\| Partly \|", roadmap, re.MULTILINE):
        failures.append(
            "docs/STARTER_BUSINESS_ROADMAP.md must keep the combined user starter at Partly "
            "until the production auth adapter is implemented"
        )

    if failures:
        print("Auth contract boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print("Auth contract boundary check passed (browser session vs Go JWT is explicit).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
