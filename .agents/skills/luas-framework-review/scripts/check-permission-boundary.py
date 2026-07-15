#!/usr/bin/env python3

"""Keep the optional permission starter fail-closed and aligned across both halves."""

from __future__ import annotations

import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]


def require_all(
    failures: list[str], relative_path: str, markers: tuple[str, ...]
) -> None:
    path = ROOT / relative_path
    if not path.exists():
        failures.append(f"{relative_path} is missing")
        return
    content = path.read_text(encoding="utf-8")
    for marker in markers:
        if marker not in content:
            failures.append(f"{relative_path} must contain {marker!r}")


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "contracts/PERMISSIONS.md",
        (
            "OPTIONAL_STARTERS=organization,permission",
            "NEXT_PUBLIC_OPTIONAL_FEATURES=organization,permission",
            "allow-only and default-deny",
            "no wildcards",
            "never directly to users",
            "subset of their own current effective permissions",
            "GET /v1/permission-context",
            "PUT /v1/organization-members/:member_id/access-roles",
            "PERMISSION.ROLE_NOT_FOUND",
            "PERMISSION.ROLE_SLUG_ALREADY_EXISTS",
            "PERMISSION.UNKNOWN",
            "Deliberate Deferrals",
        ),
    )
    require_all(
        failures,
        "api/internal/starter/assembly/starter_manifest.go",
        ("Dependencies() []string", "WithStarterDependencies"),
    )
    require_all(
        failures,
        "api/internal/starter/catalog.go",
        (
            "validateDependencies",
            "dependency cycle",
            "requires %q in OPTIONAL_STARTERS",
            "appendWithDependencies",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/permission/provider.go",
        (
            '"permission"',
            'WithStarterDependencies("organization")',
            "2026_07_15_010000_create_permission_tables",
            "wire.Bind(new(domain.PermissionAuthorizer)",
        ),
    )
    require_all(
        failures,
        "api/internal/domain/permission.go",
        (
            "type PermissionKey string",
            "PermissionKey) IsValid() bool",
            "type AccessRole struct",
            "type PermissionContext struct",
            "type PermissionAuthorizer interface",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/permission/catalog.go",
        (
            '"permission.roles.read"',
            '"permission.roles.manage"',
            '"permission.assignments.read"',
            '"permission.assignments.manage"',
            "duplicate permission key",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/permission/service.go",
        (
            "persisted permission %q is not registered",
            "permission %q is not registered",
            "withAuthorizedTransaction",
            "dominatesRoles(effective, current, &updated)",
            "dominatesRoles(effective, append(currentRoles, requestedRoles...)...)",
            "PermissionAssignmentsManage",
            "auditstarter.RecordChange",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/permission/repository.go",
        (
            "effectiveForUpdate",
            'memberships.organization_id = ? AND memberships.user_id = ?',
            'Where("organization_id = ? AND id IN ?", organizationID, roleIDs)',
            "ContextWithTransaction",
            "ErrOrganizationMemberNotFound",
            "ErrAccessRoleNotFound",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/permission/routes.go",
        (
            'WithMiddleware("auth", "organization_context")',
            'GET("/permission-context"',
            'GET("/permissions"',
            'POST("/access-roles"',
            'PUT("/organization-members/:member_id/access-roles"',
        ),
    )
    require_all(
        failures,
        "api/database/migrations/2026_07_15_010000_create_permission_tables.go",
        (
            "UseTransaction: true",
            "permission.AccessRolePO{}",
            "permission.AccessRolePermissionPO{}",
            "permission.AccessRoleAssignmentPO{}",
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
            "permission: ['organization']",
        ),
    )
    require_all(
        failures,
        "web/src/features/permission/server/permission-route.ts",
        (
            "guardSameOriginMutation(request)",
            "readJsonBody(request)",
            "organization-id",
            "forwardAuthenticatedGoApi(request",
            "PERMISSION_ROLE_NOT_FOUND",
            "PERMISSION_UNKNOWN",
        ),
    )
    require_all(
        failures,
        "web/src/features/permission/server/mock-permission-store.ts",
        (
            "hasPermission(effective, PermissionKey.ROLES_MANAGE)",
            "dominates(effective, role.permissions)",
            "touchedPermissions",
            "role_not_found",
        ),
    )
    require_all(
        failures,
        "web/src/features/permission/services/permission-service.ts",
        (
            "permissionContextSchema.safeParse",
            "accessRolePageEnvelopeSchema.safeParse",
            "memberAccessRolesSchema.safeParse",
            "'Organization-Id': String(organizationId)",
            "ClientErrorCode.INVALID_RESPONSE",
        ),
    )
    require_all(
        failures,
        "web/src/features/organization/components/organization-overview.tsx",
        (
            "isWebFeatureEnabled('permission')",
            "PermissionManagement",
            'TabsContent value="permissions"',
        ),
    )
    require_all(
        failures,
        "api/scripts/verify-compose.sh",
        (
            'permission_flow="skipped"',
            'permission_migration_flow="skipped"',
            "/app/luas db:rollback --step=1",
            "/app/luas db:migrate",
            "permission migration re-apply created",
        ),
    )

    if failures:
        print("Permission boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print("Permission boundary check passed (exact, tenant-scoped, fail-closed).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
