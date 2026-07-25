#!/usr/bin/env python3

"""Keep the default audit starter private, bounded, redacted, and retainable."""

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


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "contracts/AUDIT.md",
        (
            "durable, user-scoped history",
            "GET /v1/audit-logs",
            "returns only records whose `user_id` is the authenticated caller",
            "luas audit:prune --before=2026-04-01T00:00:00Z --batch=500",
            "must be in the past",
            "limited to 1-10,000",
            "deterministic by `(created_at, id)`",
            "does not choose a universal retention period",
            "audit-history feature or mock parity",
            "Deliberate Deferrals",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/audit/dto.go",
        (
            'binding:"omitempty,max=120"',
            'binding:"omitempty,max=180"',
            'binding:"omitempty,max=10"',
            'binding:"omitempty,max=80"',
            'binding:"omitempty,gte=100,lte=599"',
        ),
    )
    require_all(
        failures,
        "api/internal/modules/audit/service.go",
        (
            "domain.AuditLogMaintainer",
            "maxAuditPruneBatch = 10_000",
            "!before.Before(time.Now().UTC())",
            "s.repo.PruneBefore(ctx, before.UTC(), batch)",
            "redactChanges(entry.Changes)",
            "redact.Map(entry.Metadata)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/audit/repository.go",
        (
            "Where(\"user_id = ?\", userID)",
            "WITH candidates AS",
            "WHERE created_at < ?",
            "ORDER BY created_at ASC, id ASC",
            "LIMIT ?",
            "FOR UPDATE SKIP LOCKED",
            "DELETE FROM audit_logs",
            "USING candidates",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/audit/model.go",
        (
            "idx_audit_logs_created_id,priority:1",
            "idx_audit_logs_created_id,priority:2",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/audit/provider.go",
        (
            "wire.Bind(new(domain.AuditLogMaintainer)",
            'WithStarterMigrationNames("2026_07_25_000000_add_audit_retention_index")',
        ),
    )
    require_all(
        failures,
        "api/database/migrations/2026_07_25_000000_add_audit_retention_index.go",
        (
            "UseTransaction: false",
            'db.Name() != "postgres"',
            "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_created_id",
            "ON audit_logs (created_at, id)",
            "DROP INDEX CONCURRENTLY IF EXISTS idx_audit_logs_created_id",
        ),
    )
    require_all(
        failures,
        "api/internal/bootstrap/operatorcommands/audit.go",
        (
            'return "audit:prune --before=<RFC3339> [--batch=500]"',
            "defaultAuditPruneBatch = 500",
            "maxAuditPruneBatch     = 10_000",
            "auditPruneTimeout      = 30 * time.Second",
            "context.WithTimeout(ctx, auditPruneTimeout)",
            "application.AuditMaintainer.PruneAuditLogs",
            "domain.AuditActorSystem",
            'Resource:   "audit_logs"',
            '"deleted": count',
            "operator audit record failed",
        ),
    )
    require_all(
        failures,
        "api/internal/bootstrap/operatorcommands/manifest.go",
        ("NewAuditPruneCommand()",),
    )
    require_all(
        failures,
        "api/internal/app/app.go",
        ("AuditMaintainer        domain.AuditLogMaintainer",),
    )

    if failures:
        for failure in failures:
            print(f"audit boundary check failed: {failure}", file=sys.stderr)
        return 1

    print("Audit boundary check passed (private reads, redaction, bounded PostgreSQL retention).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
