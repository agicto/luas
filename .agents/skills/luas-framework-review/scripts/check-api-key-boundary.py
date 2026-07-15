#!/usr/bin/env python3
"""Guard the default API key starter's cross-service security contract."""

from __future__ import annotations

import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]


def read(relative_path: str) -> str:
    path = ROOT / relative_path
    if not path.is_file():
        raise FileNotFoundError(f"missing required file: {relative_path}")
    return path.read_text(encoding="utf-8")


def require_all(failures: list[str], relative_path: str, needles: tuple[str, ...]) -> None:
    try:
        content = read(relative_path)
    except FileNotFoundError as error:
        failures.append(str(error))
        return
    for needle in needles:
        if needle not in content:
            failures.append(f"{relative_path} must contain {needle!r}")


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "contracts/API_KEYS.md",
        (
            "The plaintext appears only in this successful",
            "JSON string array",
            "Revocation is idempotent",
            "PERMISSION.DENIED",
            "Scopes attenuate",
            "there is no rotate endpoint",
        ),
    )
    require_all(
        failures,
        "CONTEXT.md",
        (
            "**API key scope**",
            "not elevate the owning user",
            "API key scope vs role/permission",
        ),
    )

    require_all(
        failures,
        "api/internal/modules/apikey/model.go",
        ("json.Marshal(scopes)", "json.Unmarshal", "Compatibility for rows written before"),
    )
    require_all(
        failures,
        "api/internal/modules/apikey/repository.go",
        (
            "revoked_at IS NULL",
            '"last_used_at"',
            "last_used_at IS NULL OR last_used_at <= ?",
            "expires_at IS NULL OR expires_at > ?",
        ),
    )
    repository = read("api/internal/modules/apikey/repository.go")
    if re.search(r"\.Save\s*\(", repository):
        failures.append(
            "api/internal/modules/apikey/repository.go must not use full-row Save for revoke or use tracking"
        )

    require_all(
        failures,
        "api/internal/modules/apikey/service.go",
        (
            "MaxAPIKeyScopes",
            "MaxAPIKeyScopeLength",
            "apiKeyScopePattern",
            "normalizeAPIKeyScopes",
            "s.repo.Revoke",
            "s.repo.RecordUse",
            "lastUsedAtThrottle",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/apikey/scope_middleware.go",
        (
            "func RequireScopes(",
            "domain.CodePermissionDenied",
            "response.AbortUnauthorized",
            "key.HasScope(required)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/apikey/handler.go",
        ("ValidationErrorHandler", "response.HandleError(c, \"API key authentication failed\", err)"),
    )

    require_all(
        failures,
        "web/src/features/api-key/schemas.ts",
        (
            "strictObject",
            "apiKeyPageEnvelopeSchema",
            "createApiKeyResultSchema",
            "maxLength(32)",
            "apiKeyScopePattern",
        ),
    )
    require_all(
        failures,
        "web/src/features/api-key/hooks/use-api-keys.ts",
        ("gcTime: 0", "apiKeyKeys.list()", "invalidateQueries"),
    )
    require_all(
        failures,
        "web/src/features/api-key/components/api-key-panel.tsx",
        (
            "createApiKey.reset()",
            "setCreated(null)",
            "navigator.clipboard.writeText",
            "setRevokeTarget",
        ),
    )
    settings_page = read(
        "web/src/app/(protected)/(console)/console/settings/page.tsx"
    )
    if "<ApiKeyPanel />" not in settings_page or "sk_demo" in settings_page:
        failures.append(
            "Web settings must render ApiKeyPanel and must not contain a fabricated sk_demo key"
        )

    require_all(
        failures,
        "web/src/features/api-key/server/api-key-route.ts",
        (
            "resolveApiKeyRoute()",
            "guardSameOriginMutation(request)",
            "authenticateApiKeyBackend",
            "path: 'api-keys'",
            "path: `api-keys/${parsedId.data}`",
            "privateNoStoreResponse(response, ['Cookie'])",
        ),
    )
    require_all(
        failures,
        "web/src/features/api-key/server/mock-api-key-store.ts",
        (
            "createHash('sha256')",
            "key_hash:",
            "function publicApiKey",
            "owner_key",
        ),
    )
    mock_store = read("web/src/features/api-key/server/mock-api-key-store.ts")
    record_match = re.search(
        r"interface MockApiKeyRecord.*?\{(?P<body>.*?)\}", mock_store, re.DOTALL
    )
    if not record_match or "plaintext" in record_match.group("body"):
        failures.append("MockApiKeyRecord must never persist plaintext API key material")

    for relative_path in (
        "web/src/app/api/api-keys/route.ts",
        "web/src/app/api/api-keys/[apiKeyId]/route.ts",
    ):
        require_all(failures, relative_path, ("privateApiKeyResponse(",))

    for relative_path in (
        "api/internal/modules/apikey/repository_test.go",
        "api/internal/modules/apikey/handler_middleware_test.go",
        "api/internal/modules/apikey/scope_middleware_test.go",
        "web/src/test/api-key-contract.test.ts",
        "web/src/test/api-key-route.test.ts",
        "web/src/test/api-key-ui.test.tsx",
        "web/docs/API_KEYS.md",
    ):
        if not (ROOT / relative_path).is_file():
            failures.append(f"missing API key boundary evidence: {relative_path}")

    if failures:
        print("API key boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"- {failure}", file=sys.stderr)
        return 1

    print("API key boundary check passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
