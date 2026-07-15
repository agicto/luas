#!/usr/bin/env python3

"""Keep the optional setting starter finite, typed, versioned, private, and aligned."""

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


def between(content: str, start: str, end: str | None = None) -> str:
    if start not in content:
        return ""
    value = content.split(start, 1)[1]
    if end is not None and end in value:
        value = value.split(end, 1)[0]
    return value


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "contracts/SETTINGS.md",
        (
            "durable, typed overrides for a code-owned setting catalog",
            "OPTIONAL_STARTERS=organization,setting",
            "NEXT_PUBLIC_OPTIONAL_FEATURES=organization,setting",
            "`branding.display_name`",
            "`localization.locale`",
            "`localization.timezone`",
            "catalog is capped at 64 definitions",
            "arbitrary objects and arrays are not supported",
            'If-Match: "setting-v3"',
            "428 SETTING.PRECONDITION_REQUIRED",
            "412 SETTING.VERSION_CONFLICT",
            "422 SETTING.INVALID_VALUE",
            "GET /v1/settings/public",
            "GET /v1/settings/user",
            "GET /v1/organization-settings",
            "public, max-age=60, stale-while-revalidate=300",
            "private, no-store",
            "failed guard runs no cleanup",
            "defaults, enum options, request bodies, and stored JSON never enter audit changes",
            "Deliberate Deferrals",
        ),
    )
    require_all(
        failures,
        "CONTEXT.md",
        (
            "**Setting definition**",
            "**Setting override**",
            "**Effective setting**",
            "An override is not process configuration",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/setting/provider.go",
        (
            '"setting"',
            'WithStarterDependencies("user", "audit", "organization")',
            "2026_07_15_040000_create_settings_table",
            "wire.Bind(new(domain.SettingReader)",
            "wire.Bind(new(domain.AppSettingWriter)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/setting/catalog.go",
        (
            "maxSettingDefinitions = 64",
            "maxSettingValueBytes  = 4 * 1024",
            r"(?:_[a-z0-9]+)*",
            r"(?:\.[a-z][a-z0-9]*",
            '"branding.display_name"',
            '"localization.locale"',
            '"localization.timezone"',
            "NewStringDefinition(",
            "NewBooleanDefinition(",
            "NewIntegerDefinition(",
            "NewEnumDefinition(",
            "NewTimezoneDefinition(",
            "only app setting definitions may be public",
            "time.LoadLocation(name)",
            "slices.Clone(candidate.Options)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/setting/model.go",
        (
            "idx_settings_scope_subject_key",
            "settings_scope_check",
            "settings_subject_check",
            "settings_key_check",
            "settings_value_state_check",
            "settings_version_check",
            "OnDelete:CASCADE",
            "IsOverridden",
            "Version",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/setting/repository.go",
        (
            "ResolveContextDB",
            'clause.Locking{Strength: "UPDATE"}',
            "expectedVersion + 1",
            "domain.ErrSettingVersionConflict",
            '"is_overridden": false',
            '"value_json":    ""',
            "DeleteForUser",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/setting/service.go",
        (
            "ListPublicAppSettings",
            "decodeStoredSettingValue",
            "domain.ErrServiceUnavailable",
            "recordSettingAudit",
            "AccountDeletionCleanerName",
            "DeleteForUser(ctx, userID)",
        ),
    )
    try:
        service = read("api/internal/modules/setting/service.go")
    except FileNotFoundError:
        service = ""
    audit = between(service, "func recordSettingAudit(")
    if not audit:
        failures.append("setting audit metadata function is missing")
    for forbidden in ('"value"', '"default"', '"options"', "ValueJSON"):
        if forbidden in audit:
            failures.append(
                f"setting audit metadata must not contain sensitive marker {forbidden!r}"
            )

    require_all(
        failures,
        "api/internal/modules/setting/handler.go",
        (
            "http.MaxBytesReader",
            "decoder.DisallowUnknownFields()",
            "settingVersionETagPattern",
            "domain.ErrSettingPreconditionRequired",
            "public, max-age=60, stale-while-revalidate=300",
            "If-None-Match",
            "private, no-store",
            "CanManageOrganization()",
            "response.NoContent(c)",
            "// luas:bounded-list max=64 reason=finite-code-owned-catalog",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/setting/routes.go",
        (
            'r.GET("/settings/public"',
            'auth.GET("/settings/user"',
            'auth.PATCH("/settings/user/:key"',
            'auth.DELETE("/settings/user/:key"',
            'contextual.WithMiddleware("organization_context")',
            'contextual.GET("/organization-settings"',
            'contextual.PATCH("/organization-settings/:key"',
            'contextual.DELETE("/organization-settings/:key"',
        ),
    )
    require_all(
        failures,
        "api/internal/bootstrap/operatorcommands/setting.go",
        (
            'return "setting:list"',
            'return "setting:set --key=<key> --value=<json-scalar> --expected-version=<version>"',
            'return "setting:reset --key=<key> --expected-version=<version>"',
            'slices.Contains(cfg.Starters.Optional, "setting")',
            "JSON number must be an integer",
            '[]string{"KEY", "KIND", "VISIBILITY", "VERSION", "SOURCE"}',
            "recordSettingCommandAudit(",
            "domain.AuditActorSystem",
            "audit persistence failed",
        ),
    )
    require_all(
        failures,
        "api/database/migrations/2026_07_15_040000_create_settings_table.go",
        (
            "UseTransaction: true",
            "AutoMigrate(&setting.SettingPO{})",
            "DropTable(&setting.SettingPO{})",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/user/account_deletion_policy.go",
        (
            "type AccountDeletionCleaner interface",
            "RegisterCleaner",
            "Prepare evaluates every guard before running any cleanup participant",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/setting/service_test.go",
        (
            "TestServiceFailsClosedForCorruptStoredValue",
            "TestAccountDeletionCleansUserSettingsInsideTransaction",
            "TestAccountDeletionRollsBackSettingCleanupWhenLaterCleanerFails",
            "TestAccountDeletionGuardFailureLeavesSettingsUntouched",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/setting/handler_test.go",
        (
            "TestPublicSettingsUseAggregateETagAndRevalidation",
            "TestUserSettingMutationRequiresCanonicalVersionAndRejectsStaleWriter",
            "TestOrganizationMemberCanReadButCannotMutateSettings",
        ),
    )
    require_all(
        failures,
        "api/scripts/verify-compose.sh",
        (
            'setting_flow="skipped"',
            "public setting revalidation returned HTTP",
            "setting:set output exposed the setting value",
            "SETTING.PRECONDITION_REQUIRED",
            "SETTING.VERSION_CONFLICT",
            "account deletion left",
            "setting migration re-apply created",
        ),
    )
    require_all(
        failures,
        ".github/workflows/container.yml",
        ("Verify typed setting Compose lifecycle", "OPTIONAL_STARTERS: organization,setting"),
    )
    require_all(
        failures,
        "api/internal/bootstrap/operatorcommands/setting_test.go",
        (
            "TestRecordSettingCommandAuditContainsMetadataOnly",
            "TestSettingOperatorManifestRegistersAllCommands",
            'NotContains(t, recorder.entry.Metadata, "value")',
        ),
    )

    require_all(
        failures,
        "web/src/config/optional-features.ts",
        ("'setting'", "setting: ['organization']"),
    )
    require_all(
        failures,
        "web/src/features/setting/schemas.ts",
        (
            "strictObject",
            "maximum(Number.MAX_SAFE_INTEGER)",
            "maxLength(64)",
            "literal('branding.display_name')",
            "literal('localization.locale')",
            "literal('localization.timezone')",
            "literal('public')",
            "literal('private')",
            "isSupportedTimezone",
        ),
    )
    forbid_any(
        failures,
        "web/src/features/setting/schemas.ts",
        ("record(", "unknown()", "any()"),
    )
    require_all(
        failures,
        "web/src/features/setting/services/setting-service.ts",
        (
            "expectedAppSettings",
            "expectedUserSettings",
            "expectedOrganizationSettings",
            "identities.length !== expected.length",
            "ClientErrorCode.INVALID_RESPONSE",
            "'If-Match': versionETag(expectedVersion)",
            "'Organization-Id': String(organizationId)",
        ),
    )
    require_all(
        failures,
        "web/src/features/setting/server/setting-route.ts",
        (
            "resolveSettingRoute(",
            "isWebFeatureEnabled('setting')",
            "guardSameOriginMutation(request)",
            "forwardPublicGoApi(request",
            "forwardAuthenticatedGoApi(request",
            "safeConditionalHeader",
            "privateNoStoreResponse",
            "privateSettingResponse(apiJsonBodyErrorResponse(payload.error), organization)",
            "public, max-age=60, stale-while-revalidate=300",
            "SETTING_PRECONDITION_REQUIRED",
            "SETTING_INVALID_VALUE",
            "target.context?.role !== 'owner'",
            "target.context?.role !== 'admin'",
        ),
    )
    require_all(
        failures,
        "web/src/features/setting/server/mock-setting-store.ts",
        (
            "maxMockSettingSubjects = 1_000",
            "publicETag()",
            "currentVersion + 1",
            "overridden: false",
            "userStates",
            "organizationStates",
        ),
    )
    require_all(
        failures,
        "web/src/test/setting-route.test.ts",
        (
            "stable ETag and bodyless revalidation",
            "preserves user isolation, monotonic versions, stale rejection, and reset",
            "cross-origin writes before authentication or body parsing",
            "only owner/admin mutate",
            "fixed production paths and explicit conditional headers",
        ),
    )
    require_all(
        failures,
        "web/src/app/(protected)/(console)/console/settings/page.tsx",
        (
            "isWebFeatureEnabled('setting')",
            "lazy(async () =>",
            "UserSettingPanel",
            "ApiKeyPanel",
        ),
    )
    forbid_any(
        failures,
        "web/src/app/(protected)/(console)/console/settings/page.tsx",
        ("companyUrl", "supportEmail", "twoFactor", "sessionTimeout", "darkMode"),
    )
    require_all(
        failures,
        "web/src/features/setting/components/organization-setting-panel.tsx",
        ("OrganizationSettingPanel", "canManage", "localization.locale"),
    )
    require_all(
        failures,
        "web/src/test/setting-ui.test.tsx",
        (
            "typed setting preferences",
            "validated IANA timezone",
            "current version",
            "read-only for ordinary members",
            "Company URL|Support email|Two-factor|SMS|Push",
        ),
    )

    require_all(
        failures,
        "api/docs/SETTINGS.md",
        ("finite code-owned catalog", "setting:list", "## Verification"),
    )
    require_all(
        failures,
        "web/docs/SETTINGS.md",
        ("strict", "organization,setting", "## Verification"),
    )
    require_all(
        failures,
        "api/docs/adr/0007-typed-setting-boundary.md",
        ("code-owned catalog", "arbitrary JSON", "## Consequences"),
    )
    require_all(
        failures,
        "docs/STARTER_BUSINESS_ROADMAP.md",
        ("`setting` optional starter", "Yes, when enabled with `organization`", "`usage` optional starter"),
    )
    require_all(
        failures,
        ".agents/skills/downstream-app-extraction/SKILL.md",
        ("When retaining `setting`", "monotonic reset tombstones", "generic JSON editor"),
    )

    if failures:
        print("Setting boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print("Setting boundary check passed (finite, typed, versioned, privacy-aware).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
