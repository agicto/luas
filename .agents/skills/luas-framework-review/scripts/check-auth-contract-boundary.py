#!/usr/bin/env python3

"""Keep browser auth and revocable Go authentication sessions aligned."""

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


def require_missing(failures: list[str], relative_path: str) -> None:
    if (ROOT / relative_path).exists():
        failures.append(f"{relative_path} must remain removed")


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "CONTEXT.md",
        (
            "Production API adapter",
            "Authentication session",
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
            "POST /v1/logout",
            "GET /v1/users/profile",
            "opaque, cryptographically random session credential",
            "stores only its SHA-256 hash",
            "AUTH_SESSION_IDLE_TIMEOUT=168h",
            "AUTH.INVALID_CREDENTIALS",
            "per-IP and normalized/hashed per-subject buckets",
            "SERVER_TRUSTED_PROXIES",
            "## Production API Adapter",
            "API_ADAPTER_ENABLED=true",
            "API_UPSTREAM_URL=http://api:8025/v1",
            "API_UPSTREAM_MAX_RESPONSE_BYTES=1048576",
            "API_CLIENT_IP_HEADER=x-real-ip",
            "__Host-luas_auth",
            "Cache-Control: private, no-store",
            "varies on `Cookie`",
            "Sends the opaque credential to `POST /v1/logout`",
        ),
    )
    require_absent(
        failures,
        "contracts/AUTHENTICATION.md",
        (
            "does not yet ship a production adapter",
            "not ready-to-use",
            "no refresh token, token denylist, or remote logout",
        ),
    )
    require_all(
        failures,
        "contracts/AUTHENTICATION.md",
        (
            "## Static SPA Authentication Boundary",
            "`web-spa/` has no server runtime",
            "same-origin browser gateway",
            "must not be stored in `localStorage`, `sessionStorage`, IndexedDB",
            "protected SPA authentication is deliberately incomplete",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/user/routes.go",
        (
            'r.POST("/login"',
            'r.POST("/register"',
            'auth.POST("/logout"',
            'auth.GET("/users/profile"',
            "sessionAuth",
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
        (
            'json:"username"',
            'json:"access_token"',
            'json:"token_type"',
            'json:"expires_in"',
            'json:"user"',
        ),
    )
    require_all(
        failures,
        "api/internal/modules/user/model.go",
        (
            "type AuthenticationSessionPO struct",
            "TokenHash",
            'gorm:"size:64;not null;uniqueIndex"',
            "IdleExpiresAt",
            "RevokedAt",
        ),
    )
    require_absent(
        failures,
        "api/internal/modules/user/model.go",
        ("IPAddress", "UserAgent", "PlaintextToken"),
    )
    require_all(
        failures,
        "api/internal/modules/user/session_service.go",
        (
            "authenticationCredentialBytes   = 32",
            "crypto.GenerateKey",
            "crypto.SHA256Hex(credential)",
            "func (s *SessionService) Authenticate",
            "PruneAuthenticationSessions",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/user/repository.go",
        (
            "UpdatePasswordAndRevokeSessions",
            "sessionRevocationPasswordReset",
            "sessionRevocationAccountDeleted",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/user/authentication_middleware.go",
        ("sessionAuth", "domain.SessionAuthenticator", "authenticationSessionID"),
    )
    require_all(
        failures,
        "api/database/migrations/2026_04_27_000003_create_authentication_sessions_table.go",
        ("AuthenticationSessionPO", "UseTransaction: true", "DropTable"),
    )
    require_all(
        failures,
        "api/internal/bootstrap/operatorcommands/authentication_session.go",
        ("auth-session:prune", "PruneAuthenticationSessions", "--batch"),
    )
    require_all(
        failures,
        "api/.env.example",
        (
            "AUTH_SESSION_TTL=720h",
            "AUTH_SESSION_IDLE_TIMEOUT=168h",
            "AUTH_SESSION_TOUCH_INTERVAL=5m",
            "AUTH_SESSION_RETENTION=720h",
        ),
    )
    if re.search(r"^JWT_(?:SECRET|EXPIRE_DAYS)=", env_example, re.MULTILINE):
        failures.append("api/.env.example must not advertise retired JWT configuration")
    require_missing(failures, "api/internal/infra/jwt/jwt.go")
    require_missing(failures, "api/internal/infra/middleware/jwt.go")
    require_missing(failures, "web/src/features/auth/server/auth-token.ts")
    if "github.com/golang-jwt/jwt" in read("api/go.mod"):
        failures.append("api/go.mod must not retain the removed JWT runtime dependency")
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
            "API_ADAPTER_ENABLED",
            "API_UPSTREAM_TIMEOUT_MS",
            "API_UPSTREAM_MAX_RESPONSE_BYTES",
            "API_UPSTREAM_URL",
            "API_CLIENT_IP_HEADER",
            "environmentAlias(",
            "AUTH_ADAPTER_ENABLED",
            "AUTH_API_URL",
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
        "web/src/features/auth/server/auth-credential.ts",
        (
            "isOpaqueAuthenticationCredential",
            "{43}",
            "byteLength === 32",
            "authenticationSessionMaxAgeSeconds",
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
            "'logout'",
            "value.token_type !== 'Bearer'",
            "value.expires_in",
            "new GoApiClient(",
            "generatedUsername",
            "mapGoApiUser",
        ),
    )
    require_all(
        failures,
        "web/src/server/api-adapter/go-api-client.ts",
        (
            "cache: 'no-store'",
            "redirect: 'error'",
            "headers.set('authorization'",
            "candidate.includes(',')",
            "readBoundedBody(",
            "maxResponseBytes",
        ),
    )
    require_all(
        failures,
        "web/src/features/auth/server/auth-response.ts",
        ("privateAuthResponse<", "privateNoStoreResponse(response, ['Cookie'])"),
    )
    require_all(
        failures,
        "web/src/features/auth/server/auth-adapter-route.ts",
        (
            "setApiSessionCookie",
            "getApiSessionToken",
            "clearApiSessionCookie",
            "configuredAdapter().logout",
            "resolveGoApiAuthBootstrap",
            "privateAuthResponse(",
        ),
    )
    require_all(
        failures,
        "web/src/server/http/private-response.ts",
        (
            "private, no-store",
            "privateNoStoreHeaders(",
            "privateNoStoreResponse<",
            "current === '*'",
            "value.toLowerCase() === name.toLowerCase()",
        ),
    )
    for route in ("login", "register", "me", "logout"):
        require_all(
            failures,
            f"web/src/app/api/auth/{route}/route.ts",
            ("resolveAuthRoute()", "privateAuthResponse("),
        )
    for test_file in (
        "web/src/test/auth-adapter-route.test.ts",
        "web/src/test/auth-route-backend.test.ts",
        "web/src/test/auth-session-cookie.test.ts",
        "web/src/test/go-api-auth-adapter.test.ts",
        "web/src/test/private-response.test.ts",
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
            "opaque credential",
            "`POST /v1/logout`",
            "Cache-Control: private, no-store",
            "src/server/http/private-response.ts",
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
    require_all(
        failures,
        "web-spa/docs/SECURITY.md",
        (
            "Every byte under `dist/` is public",
            "reviewed browser gateway or Go adapter",
            "HttpOnly session cookie",
            "Do not put its `access_token` in `localStorage`, `sessionStorage`, IndexedDB",
            "protected SPA auth is not",
            "A client-side route guard",
        ),
    )
    require_all(
        failures,
        "web-spa/src/http/client.ts",
        (
            "credentials: 'include'",
            "HttpStatusErrorCodeMap[response.status]",
            "ClientErrorCode.TIMEOUT",
            "ClientErrorCode.NETWORK_ERROR",
        ),
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

    print("Auth contract boundary check passed (revocable sessions and adapter ownership are explicit).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
