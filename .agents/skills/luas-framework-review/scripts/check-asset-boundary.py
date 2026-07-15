#!/usr/bin/env python3

"""Keep the optional asset starter private, bounded, provider-neutral, and aligned."""

from __future__ import annotations

import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]


def read(relative_path: str) -> str:
    path = ROOT / relative_path
    if not path.exists():
        raise FileNotFoundError(relative_path)
    return path.read_text(encoding="utf-8")


def require_all(
    failures: list[str], relative_path: str, markers: tuple[str, ...]
) -> None:
    try:
        content = read(relative_path)
    except FileNotFoundError:
        failures.append(f"{relative_path} is missing")
        return
    for marker in markers:
        if marker not in content:
            failures.append(f"{relative_path} must contain {marker!r}")


def forbid_any(
    failures: list[str], relative_path: str, markers: tuple[str, ...]
) -> None:
    try:
        content = read(relative_path)
    except FileNotFoundError:
        failures.append(f"{relative_path} is missing")
        return
    for marker in markers:
        if marker in content:
            failures.append(f"{relative_path} must not contain {marker!r}")


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "contracts/ASSETS.md",
        (
            "OPTIONAL_STARTERS=asset",
            "NEXT_PUBLIC_OPTIONAL_FEATURES=asset",
            "An **asset** is the user-owned metadata and lifecycle record",
            "A **stored object** is the opaque byte",
            "pending -> ready",
            "random staging key",
            "frozen final snapshot",
            "10 MiB",
            "GET /v1/assets",
            "POST /v1/assets/upload-intents",
            "POST /v1/assets/:id/complete",
            "POST /v1/assets/:id/download-grant",
            "DELETE /v1/assets/:id",
            "HTTP 204 No Content",
            "ASSET.NOT_FOUND",
            "ASSET.INVALID_MEDIA_TYPE",
            "account-deletion guard",
            "tombstone retains its private cleanup keys",
            "Deliberate Deferrals",
        ),
    )
    require_all(
        failures,
        "api/internal/capabilities/storage/object_store.go",
        (
            "type ObjectStore interface",
            "type DirectTransferStore interface",
            "URL is a bearer credential",
            "Business modules own metadata and lifecycle",
            "func ValidateObjectKey",
            'segment == ".."',
        ),
    )
    require_all(
        failures,
        "api/internal/infra/storage/local.go",
        (
            "os.OpenRoot",
            "root.MkdirAll(directory, 0o700)",
            "os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600",
            "io.LimitReader(body, size+1)",
            "root.Rename(temporaryKey, key)",
            "storagecap.ValidateObjectKey(key)",
        ),
    )
    forbid_any(
        failures,
        "api/internal/infra/storage/local.go",
        (
            "filepath.Join(s.root, key)",
            "os.WriteFile(",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/storage/r2.go",
        (
            '"github.com/aws/aws-sdk-go-v2/',
            "UsePathStyle = true",
            "SwapComputePayloadSHA256ForUnsignedPayloadMiddleware",
            "PresignPutObject",
            "PresignGetObject",
            "browserMaySetHeader",
            'case "authorization", "content-length", "cookie", "host", "origin", "referer", "set-cookie"',
            "attachmentDisposition",
            "return storagecap.ErrObjectStoreUnavailable",
        ),
    )
    forbid_any(
        failures,
        "api/go.mod",
        (
            "github.com/aws/aws-sdk-go v1",
            "github.com/aws/aws-sdk-go/",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/config/config.go",
        (
            "type ObjectStorageConfig struct",
            "type AssetConfig struct",
            "OBJECT_STORAGE_DRIVER must be r2 for the asset starter in production",
            "OBJECT_STORAGE_REQUEST_TIMEOUT",
            "ASSET_MAX_BYTES must be between 1 and 104857600",
            "ASSET_PENDING_TTL must be at least ASSET_UPLOAD_GRANT_TTL",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/asset/model.go",
        (
            "idx_assets_user_idempotency",
            "idx_assets_cleanup",
            "assets_status_check",
            "OperationToken",
            "PendingExpiresAt",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/asset/provider.go",
        (
            '"asset"',
            'WithStarterDependencies("user", "audit")',
            "2026_07_15_030000_create_assets_table",
            "wire.Bind(new(domain.AssetReader)",
            "wire.Bind(new(domain.AssetMaintainer)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/asset/service.go",
        (
            '"asset-uploads/" + id + "/object"',
            '"assets/" + id + "/object"',
            "ClaimCompletion",
            "s.inspector.Inspect",
            "s.objects.Copy",
            "Freeze staging bytes under the immutable final key before inspection",
            "MarkReady",
            "ClaimDeletion",
            "CountActiveForUser",
            "func (s *service) Prune",
            "recordAssetAudit",
            "context.WithoutCancel(ctx)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/asset/repository.go",
        (
            "ResolveContextDB",
            'clause.Locking{Strength: "UPDATE"}',
            "pending_expires_at <= ?",
            "domain.AssetStatusDeleted",
            "preserveFinal bool",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/asset/service_test.go",
        (
            "TestServiceInspectsFrozenFinalObjectAfterStagingChanges",
            "TestServicePrunesLateReadyStagingWithoutDeletingFinalObject",
            "TestServicePrunesLateDeletedStagingAfterGrantExpiry",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/asset/routes.go",
        (
            'auth.GET("/assets"',
            'auth.POST("/assets/upload-intents"',
            'auth.POST("/assets/:id/complete"',
            'auth.POST("/assets/:id/download-grant"',
            'auth.DELETE("/assets/:id"',
            'r.PUT("/asset-transfers/:token"',
            'r.GET("/asset-transfers/:token"',
        ),
    )
    require_all(
        failures,
        "api/internal/modules/asset/dto.go",
        (
            'json:"original_name"',
            'json:"media_type"',
            'json:"size_bytes"',
            'json:"status"',
            'json:"ready_at"',
        ),
    )
    require_all(
        failures,
        "api/internal/modules/asset/handler.go",
        ("func (h *Handler) Delete", "response.NoContent(c)"),
    )
    require_all(
        failures,
        "api/internal/modules/asset/handler_test.go",
        ("TestHandlerDeleteReturnsNoContent", "http.StatusNoContent"),
    )
    forbid_any(
        failures,
        "api/internal/modules/asset/dto.go",
        (
            'json:"user_id"',
            'json:"object_key"',
            'json:"staging_key"',
            'json:"checksum',
        ),
    )
    require_all(
        failures,
        "api/internal/infra/console/commands/asset.go",
        (
            'return "asset:prune"',
            'return "asset:prune [--batch=100]"',
            'slices.Contains(cfg.Starters.Optional, "asset")',
            "runAssetPrune",
        ),
    )
    require_all(
        failures,
        "api/database/migrations/2026_07_15_030000_create_assets_table.go",
        (
            "UseTransaction: true",
            "AutoMigrate(&asset.AssetPO{})",
            "DropTable(&asset.AssetPO{})",
            "remove provider objects before rollback",
        ),
    )

    require_all(
        failures,
        "web/src/features/asset/schemas.ts",
        (
            "strictObject",
            "ASSET_MAX_BROWSER_BYTES",
            "transferGrantSchema",
            "uploadIntentResultSchema",
        ),
    )
    forbid_any(
        failures,
        "web/src/features/asset/schemas.ts",
        ("deleteAssetResultSchema",),
    )
    require_all(
        failures,
        "web/src/features/asset/services/asset-transfer.ts",
        (
            "localTransferPath",
            "credentials: 'omit'",
            "cache: 'no-store'",
            "redirect: 'error'",
            "referrerPolicy: 'no-referrer'",
            "url.protocol !== 'https:'",
            "URL.revokeObjectURL",
            "forbiddenHeaders",
            "readBoundedBlob",
            "received > expectedBytes",
            "content-length",
        ),
    )
    require_all(
        failures,
        "web/src/features/asset/services/asset-service.ts",
        (
            "parseUploadIntent",
            "await uploadToGrant(intent.upload, file)",
            "return this.complete(intent.asset.id)",
            "parseTransferGrant",
            "downloadFromGrant",
            "await request.delete<void>",
        ),
    )
    require_all(
        failures,
        "web/src/features/asset/server/asset-route.ts",
        (
            "resolveAssetRoute(",
            "isWebFeatureEnabled('asset')",
            "guardSameOriginMutation(request)",
            "privateNoStoreResponse(response, ['Cookie'])",
            "forwardAuthenticatedGoApi(request",
            "readJsonBody(request, maxIntentBodyBytes)",
            "mockAssetStore.acceptUpload",
            "apiNoContentResponse()",
        ),
    )
    require_all(
        failures,
        "web/src/features/asset/server/mock-asset-store.ts",
        (
            "maxMockBytesPerUser",
            "maxAssetsPerUser",
            "maxTransferTokens",
            "pruneExpiredTokens",
            "readExactBody",
            "createHash('sha256')",
            "TextDecoder('utf-8', { fatal: true })",
            "'content-disposition'",
            "'cache-control': 'private, no-store'",
        ),
    )
    require_all(
        failures,
        "web/src/features/asset/components/asset-panel.tsx",
        (
            "export function AssetPanel",
            "useUploadAsset",
            "useDownloadAsset",
            "useDeleteAsset",
            "Dialog",
            "Table",
        ),
    )
    require_all(
        failures,
        "web/src/config/optional-features.ts",
        (
            "OPTIONAL_WEB_FEATURES = [",
            "'organization'",
            "'permission'",
            "'notification'",
            "'asset'",
        ),
    )

    require_all(
        failures,
        "api/docs/ASSETS.md",
        (
            "asset",
            "stored object",
            "production",
            "R2_SECRET_ACCESS_KEY",
            "## Verification",
        ),
    )
    require_all(
        failures,
        "web/docs/ASSETS.md",
        ("asset", "stored object", "production", "## Verification"),
    )
    require_all(
        failures,
        "api/docs/adr/0006-asset-object-boundary.md",
        ("asset", "stored object", "Production activation requires R2", "## Consequences"),
    )
    require_all(
        failures,
        "docs/STARTER_BUSINESS_ROADMAP.md",
        ("`asset` optional starter", "Yes, when enabled", "Build durable typed `setting` next"),
    )
    require_all(
        failures,
        "api/scripts/verify-compose.sh",
        (
            'asset_flow="skipped"',
            'asset_migration_flow="skipped"',
            "asset upload intent returned HTTP",
            "asset download bytes differ from uploaded bytes",
            "asset migration re-apply created",
            'asset_delete_status}" == "204"',
            'asset_account_race_flow="skipped"',
            '!= "201/409"',
            '!= "404/204"',
            "orphaned_assets",
        ),
    )
    forbid_any(
        failures,
        "api/scripts/verify-compose.sh",
        ('!= "404/200"',),
    )
    require_all(
        failures,
        "web/src/test/asset-contract.test.ts",
        ("asset browser contract", "unknown intent fields", "unsafe requests"),
    )
    require_all(
        failures,
        "web/src/test/asset-route.test.ts",
        ("asset browser route boundary", "cross-origin", "mockUser('bob')"),
    )
    require_all(
        failures,
        "web/src/test/asset-ui.test.tsx",
        ("asset console workflow", "unsupported files", "requires confirmation"),
    )

    if failures:
        print("Asset boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print("Asset boundary check passed (private, bounded, provider-neutral).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
