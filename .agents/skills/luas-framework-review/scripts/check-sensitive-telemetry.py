#!/usr/bin/env python3
"""Guard Luas request telemetry, diagnostics, SQL spans, and audits from secret leakage."""

from __future__ import annotations

import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]


def read(relative_path: str) -> str:
    path = ROOT / relative_path
    if not path.is_file():
        raise FileNotFoundError(f"missing required file: {relative_path}")
    return path.read_text(encoding="utf-8")


def require_all(failures: list[str], relative_path: str, needles: tuple[str, ...]) -> str:
    try:
        content = read(relative_path)
    except FileNotFoundError as error:
        failures.append(str(error))
        return ""
    for needle in needles:
        if needle not in content:
            failures.append(f"{relative_path} must contain {needle!r}")
    return content


def forbid_all(failures: list[str], relative_path: str, needles: tuple[str, ...]) -> None:
    try:
        content = read(relative_path)
    except FileNotFoundError as error:
        failures.append(str(error))
        return
    for needle in needles:
        if needle in content:
            failures.append(f"{relative_path} must not contain {needle!r}")


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "api/pkg/redact/redact.go",
        (
            'Placeholder = "[REDACTED]"',
            "func IsSensitiveKey(",
            "func Map(",
            "func Headers(",
            "func Query(",
            "func URLWithPath(",
        ),
    )
    for relative_path in (
        "api/pkg/logger/gin.go",
        "api/internal/infra/middleware/logger.go",
    ):
        require_all(failures, relative_path, ("c.FullPath()", 'path = "unmatched"'))
        forbid_all(
            failures,
            relative_path,
            ("RawQuery", "URL.String()", "c.Request.URL.Path"),
        )

    require_all(
        failures,
        "api/pkg/logger/core.go",
        ('"github.com/zgiai/luas/api/pkg/redact"', "return redact.Map(merged)"),
    )
    require_all(
        failures,
        "api/internal/infra/exception/collector.go",
        ("redact.Headers(req.Header)", "redact.Query(req.URL.Query())"),
    )
    require_all(
        failures,
        "api/internal/infra/exception/recovery.go",
        (
            "redact.URLWithPath",
            "json.MarshalIndent(redact.Map(ctx)",
            "path := c.FullPath()",
            'path = "unmatched"',
        ),
    )
    require_all(
        failures,
        "api/pkg/errors/handler.go",
        (
            "redact.URLWithPath",
            "escapeDebugPageData",
            "redactDebugRequest",
            "html.EscapeString",
            'logger.Error("HTTP Request Error"',
        ),
    )
    forbid_all(
        failures,
        "api/pkg/errors/handler.go",
        ("fmt.Println", "c.Errors.String()"),
    )
    require_all(
        failures,
        "api/pkg/logger/gin.go",
        ('fields["error_count"]', 'fields["error_types"]'),
    )
    forbid_all(failures, "api/pkg/logger/gin.go", ("c.Errors.String()",))
    require_all(
        failures,
        "api/internal/infra/database/database.go",
        ("ParameterizedQueries:      true",),
    )
    require_all(
        failures,
        "api/internal/infra/database/observed_logger.go",
        (
            "func (l *observedLogger) ParamsFilter",
            "logger.RecorderParamsFilter = parameterizedQueryFilter",
            "return sql, nil",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/tracing/middleware.go",
        (
            "semconv.HTTPRoute(route)",
            'attribute.Int("http.error_count"',
            'attribute.StringSlice("error.types"',
        ),
    )
    forbid_all(
        failures,
        "api/internal/infra/tracing/middleware.go",
        ("semconv.URLPath", "c.Errors.String()", "span.RecordError"),
    )
    require_all(
        failures,
        "api/internal/infra/tracing/tracing.go",
        ("func recordErrorType", "semconv.ErrorTypeKey", 'span.SetStatus(codes.Error, "")'),
    )
    require_all(
        failures,
        "api/internal/infra/tracing/gorm.go",
        ("recordErrorType(span, db.Error)",),
    )
    forbid_all(
        failures,
        "api/internal/infra/tracing/gorm.go",
        ("span.RecordError", "db.Error.Error()"),
    )
    require_all(
        failures,
        "api/internal/modules/audit/context.go",
        ("redactChanges", "redact.IsSensitiveKey", "redact.Map(change.Metadata)"),
    )
    require_all(
        failures,
        "api/internal/modules/audit/service.go",
        ("entry.Changes = redactChanges", "entry.Metadata = redact.Map"),
    )

    for relative_path in (
        "api/pkg/redact/redact_test.go",
        "api/pkg/logger/gin_test.go",
        "api/internal/infra/exception/recovery_test.go",
        "api/internal/infra/database/database_test.go",
        "api/internal/infra/tracing/middleware_test.go",
        "api/internal/modules/audit/service_test.go",
        "api/docs/OBSERVABILITY.md",
    ):
        if not (ROOT / relative_path).is_file():
            failures.append(f"missing sensitive telemetry evidence: {relative_path}")

    if failures:
        print("Sensitive telemetry boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"- {failure}", file=sys.stderr)
        return 1

    print("Sensitive telemetry boundary check passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
