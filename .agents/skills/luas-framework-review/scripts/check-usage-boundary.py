#!/usr/bin/env python3

"""Keep trusted usage metering finite, atomic, private, and contract-aligned."""

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
        "contracts/USAGE.md",
        (
            "optional `usage` starter",
            "OPTIONAL_STARTERS=organization,usage",
            "NEXT_PUBLIC_OPTIONAL_FEATURES=organization,usage",
            "catalog is capped at 64 scoped definitions",
            "each with a finite enum allowlist",
            "at most 32 allowed values",
            "`source + event_id` is globally unique",
            "9007199254740991",
            "`RecordUsage`",
            "`ConsumeUsage`",
            "24 hours old",
            "5 minutes in the future",
            "GET` | `/v1/usage/user`",
            "GET` | `/v1/organization-usage`",
            "private, no-store",
            "minimum 90-day idempotency horizon",
            "usage:quota:set",
            "usage:prune",
            "Deliberate Deferrals",
        ),
    )
    require_all(
        failures,
        "CONTEXT.md",
        (
            "**Usage metric**",
            "**Usage event**",
            "**Usage counter**",
            "**Usage quota**",
            "**usage vs telemetry/rate limit/billing**",
        ),
    )
    require_all(
        failures,
        "api/internal/domain/usage.go",
        (
            "type UsageReader interface",
            "type UsageRecorder interface",
            "type UsageConsumer interface",
            "type UsageQuotaWriter interface",
            "type UsageMaintainer interface",
            "RecordUsage(ctx context.Context",
            "ConsumeUsage(ctx context.Context",
            "expectedVersion uint64",
        ),
    )
    require_all(
        failures,
        "api/internal/domain/error_codes.go",
        (
            '"USAGE.METRIC_NOT_FOUND"',
            '"USAGE.INVALID_EVENT"',
            '"USAGE.EVENT_OUTSIDE_WINDOW"',
            '"USAGE.IDEMPOTENCY_CONFLICT"',
            '"USAGE.QUOTA_EXCEEDED"',
            '"USAGE.QUOTA_VERSION_CONFLICT"',
            '"USAGE.PRECONDITION_REQUIRED"',
        ),
    )
    require_all(
        failures,
        "api/internal/bootstrap/domain_error_mappings.go",
        (
            "ErrUsageMetricNotFound, http.StatusNotFound",
            "ErrUsageIdempotencyConflict, http.StatusConflict",
            "ErrUsageQuotaVersionConflict, http.StatusPreconditionFailed",
            "ErrUsageInvalidEvent, http.StatusUnprocessableEntity",
            "ErrUsageEventOutsideWindow, http.StatusUnprocessableEntity",
            "ErrUsagePreconditionRequired, http.StatusPreconditionRequired",
            "ErrUsageQuotaExceeded, http.StatusTooManyRequests",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/usage/provider.go",
        (
            '"usage"',
            'WithStarterDependencies("user", "audit", "organization")',
            "2026_07_15_050000_create_usage_tables",
            "wire.Bind(new(domain.UsageReader)",
            "wire.Bind(new(domain.UsageRecorder)",
            "wire.Bind(new(domain.UsageConsumer)",
            "wire.Bind(new(domain.UsageQuotaWriter)",
            "wire.Bind(new(domain.UsageMaintainer)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/usage/catalog.go",
        (
            "maxUsageDefinitions     = 64",
            "maxUsageDimensions      = 8",
            "maxUsageDimensionValues = 32",
            "maxSafeUsageInteger     = int64(9_007_199_254_740_991)",
            '"api.requests"',
            '"ai.input_tokens"',
            '"ai.output_tokens"',
            '"asset.transfer_bytes"',
            '"workflow.runs"',
            "domain.UsageScopeUser",
            "domain.UsageScopeOrganization",
            "normalizeUsageDimensions(",
            "cloneDimensionDefinitions(candidate.Dimensions)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/usage/model.go",
        (
            "idx_usage_events_source_event",
            "usage_events_quantity_check",
            "usage_events_decision_check",
            "idx_usage_counters_identity",
            "usage_counters_value_check",
            "idx_usage_quotas_identity",
            "usage_quotas_state_check",
            "9007199254740991",
            "OnDelete:CASCADE",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/usage/repository.go",
        (
            "ResolveContextDB",
            "clause.OnConflict{DoNothing: true}",
            "Fingerprint != mutation.Fingerprint",
            "lockUsageOwner(tx, mutation.Target)",
            'clause.Locking{Strength: "NO KEY UPDATE"}',
            "findUsageCounterForUpdate(",
            'clause.Locking{Strength: "UPDATE"}',
            "decision = domain.UsageDecisionDenied",
            "after = before",
            'Where("id = ? AND version = ?"',
            '"is_overridden": false',
            "PruneReceipts",
            "decision <> ?",
            "DeleteForUser",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/usage/service.go",
        (
            "usageMaxEventAge      = 24 * time.Hour",
            "usageMaxFutureSkew    = 5 * time.Minute",
            "usageReceiptRetention = 90 * 24 * time.Hour",
            "func (s *service) RecordUsage(",
            "func (s *service) ConsumeUsage(",
            "domain.ErrUsageQuotaExceeded",
            "domain.ErrServiceUnavailable",
            "AccountDeletionCleanerName() string { return \"usage\" }",
            "DeleteForUser(ctx, userID)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/usage/handler.go",
        (
            "CanManageOrganization()",
            "private, no-store",
            'c.Header("Pragma", "no-cache")',
            'c.Header("Vary", "Authorization, Organization-Id")',
            "// luas:bounded-list max=64 reason=finite-code-owned-catalog",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/usage/routes.go",
        (
            'auth.GET("/usage/user"',
            'contextual.WithMiddleware("organization_context")',
            'contextual.GET("/organization-usage"',
        ),
    )
    forbid_any(
        failures,
        "api/internal/modules/usage/routes.go",
        (".POST(", ".PUT(", ".PATCH(", ".DELETE("),
    )
    require_all(
        failures,
        "api/database/migrations/2026_07_15_050000_create_usage_tables.go",
        (
            "UseTransaction: true",
            "UsageCounterPO{}",
            "UsageQuotaPO{}",
            "UsageEventPO{}",
            "DropTable(",
        ),
    )
    require_all(
        failures,
        "api/internal/bootstrap/operatorcommands/usage.go",
        (
            'return "usage:list',
            'return "usage:record',
            'return "usage:consume',
            'return "usage:quota:set',
            'return "usage:quota:reset',
            'return "usage:prune',
            'slices.Contains(cfg.Starters.Optional, "usage")',
            "recordUsageOperatorAudit(",
            "recordUsageQuotaOperatorAudit(",
            "recordUsagePruneAudit(",
            "audit persistence failed",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/usage/service_test.go",
        (
            "TestUsageRecordIsIdempotentAndSupportsBoundedCorrections",
            "TestUsageConsumeAtomicallyPersistsAcceptedAndDeniedDecisions",
            "TestUsageQuotaCASRetainsResetHistoryAndRecordCanExposeOverage",
            "TestUsageFailsClosedForCorruptQuotaAndCounterState",
            "TestUsageRetentionAndAccountDeletionPreserveOwnedBoundaries",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/usage/handler_test.go",
        (
            "TestUsageUserListReturnsFinitePrivateSummaryOnly",
            "TestUsageOrganizationListRequiresManagerRoleAndPrivateContext",
        ),
    )
    require_all(
        failures,
        "api/scripts/verify-compose.sh",
        (
            'usage_flow="skipped"',
            "concurrent usage quota expected one success and three denials",
            "usage migration rollback failed",
            "usage migration re-apply created",
            "usage account cleanup fixture has no owned rows",
        ),
    )
    require_all(
        failures,
        ".github/workflows/container.yml",
        (
            "Verify usage metering Compose lifecycle",
            "OPTIONAL_STARTERS: organization,usage",
        ),
    )

    require_all(
        failures,
        "web/src/config/optional-features.ts",
        ("'usage'", "usage: ['organization']"),
    )
    require_all(
        failures,
        "web/src/features/usage/schemas.ts",
        (
            "strictObject",
            "maximum(Number.MAX_SAFE_INTEGER)",
            "maxLength(64)",
            "literal('api.requests')",
            "literal('ai.input_tokens')",
            "literal('ai.output_tokens')",
            "literal('asset.transfer_bytes')",
            "literal('workflow.runs')",
            "isSemanticallyValidSummary",
        ),
    )
    forbid_any(
        failures,
        "web/src/features/usage/schemas.ts",
        ("record(", "unknown()", "any()"),
    )
    require_all(
        failures,
        "web/src/features/usage/services/usage-service.ts",
        (
            "expectedMetrics",
            "usageSummaryListSchema.safeParse(value)",
            "ClientErrorCode.INVALID_RESPONSE",
            "'/usage/user'",
            "'/organization-usage'",
            "'Organization-Id': String(organizationId)",
        ),
    )
    require_all(
        failures,
        "web/src/features/usage/server/usage-route.ts",
        (
            "resolveUsageRoute(",
            "isWebFeatureEnabled('usage')",
            "forwardAuthenticatedGoApi(request",
            "path: 'usage/user'",
            "path: 'organization-usage'",
            "privateNoStoreHeaders(",
            "target.context?.role !== 'owner'",
            "target.context?.role !== 'admin'",
        ),
    )
    require_all(
        failures,
        "web/src/features/usage/server/mock-usage-store.ts",
        (
            "const metricCatalog = [",
            "user(_user: AuthUser",
            "organization(_organizationId: number",
            "quota_source:",
        ),
    )
    require_all(
        failures,
        "web/src/app/api/usage/user/route.ts",
        ("export async function GET", "privateUsageResponse"),
    )
    require_all(
        failures,
        "web/src/app/api/organization-usage/route.ts",
        ("export async function GET", "privateUsageResponse"),
    )
    forbid_any(
        failures,
        "web/src/app/api/usage/user/route.ts",
        ("function POST", "function PUT", "function PATCH", "function DELETE"),
    )
    forbid_any(
        failures,
        "web/src/app/api/organization-usage/route.ts",
        ("function POST", "function PUT", "function PATCH", "function DELETE"),
    )
    require_all(
        failures,
        "web/src/test/usage-service.test.ts",
        (
            "exact finite user and organization catalogs",
            "unknown metrics",
            "unsafe integers",
        ),
    )
    require_all(
        failures,
        "web/src/test/usage-route.test.ts",
        (
            "private user usage",
            "organization managers",
            "fixed production paths",
        ),
    )

    require_all(
        failures,
        "api/docs/USAGE.md",
        ("UsageRecorder", "ConsumeUsage", "usage:prune", "## Verification"),
    )
    require_all(
        failures,
        "web/docs/USAGE.md",
        ("read-only", "organization,usage", "## Verification"),
    )
    require_all(
        failures,
        "api/docs/adr/0008-usage-metering-and-quota-boundary.md",
        ("code-owned", "There is no public or browser write endpoint", "## Consequences"),
    )
    require_all(
        failures,
        "docs/STARTER_BUSINESS_ROADMAP.md",
        (
            "`usage` optional starter",
            "Yes, when enabled with `organization`",
            "Build `webhook` next",
        ),
    )
    require_all(
        failures,
        ".agents/skills/downstream-app-extraction/SKILL.md",
        ("When retaining `usage`", "stable `source + event_id`"),
    )

    if failures:
        print("usage boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"- {failure}", file=sys.stderr)
        return 1

    print("usage boundary check passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
