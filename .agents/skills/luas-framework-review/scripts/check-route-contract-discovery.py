#!/usr/bin/env python3

"""Keep route discovery attached to runtime assembly and honest contract semantics."""

from __future__ import annotations

import json
import stat
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
            failures.append(f"{relative_path} must not contain {marker!r}")


def require_executable(failures: list[str], relative_path: str) -> None:
    path = ROOT / relative_path
    if not path.exists():
        failures.append(f"{relative_path} is missing")
        return
    if not path.stat().st_mode & stat.S_IXUSR:
        failures.append(f"{relative_path} must be executable")


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "api/internal/bootstrap/http.go",
        (
            "func RegisterHTTPRoutes(",
            "h.RegisterRoutes(engine)",
            'engine.GET("/metrics", metrics.Handler())',
            "routes.Setup(engine, application.Starters)",
            "h := RegisterHTTPRoutes(r, application)",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/console/commands/route_list.go",
        (
            'routeCatalogKind          = "luas.route_catalog"',
            "routeCatalogSchemaVersion = 1",
            "bootstrap.RegisterHTTPRoutes(engine, application)",
            'json:"schema_version"',
            'json:"active_starters"',
            'json:"routes"',
            "SuppressCompletionOutput() bool",
            "sort.Slice(registered",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/console/console.go",
        ("completionOutputSuppressor", "shouldPrintCompletionOutput(cmd)"),
    )
    require_all(
        failures,
        "api/scripts/verify-route-catalog.sh",
        (
            'go build -o "${CLI_FILE}" ./cmd/luas',
            '"${CLI_FILE}" route:list --format=json',
            "validate-route-catalog.py",
            "--expect-starter audit",
            "--expect-starter apikey",
            "--expect-starter user",
            "--require-route GET /health/ready",
            "--require-route POST /v1/login",
        ),
    )
    require_all(
        failures,
        "api/docs/ROUTE_DISCOVERY.md",
        (
            "One runtime registration seam",
            "not an OpenAPI Description",
            "route-catalog.schema.json",
            "make route-catalog-check",
        ),
    )
    require_all(
        failures,
        "contracts/README.md",
        (
            "## Contract Discovery",
            "route:list --format=json",
            "OpenAPI 3.1 machine contract",
            "does not infer payloads",
        ),
    )
    require_all(
        failures,
        "Makefile",
        ("check-route-contract-discovery.py", "make route-catalog-check"),
    )
    require_all(
        failures,
        "api/Makefile",
        ("route-catalog-check:", "scripts/verify-route-catalog.sh"),
    )
    require_all(
        failures,
        ".github/workflows/ci.yml",
        ("Verify runtime route catalog", "make route-catalog-check"),
    )
    require_absent(
        failures,
        "web/src/app/(protected)/(console)/console/page.tsx",
        ("OpenAPI", "LUAS_CONTRACTS_URL", "github.com/zgiai/luas/tree/main/contracts"),
    )

    for relative_path in (
        "api/cmd/tools/routes",
        "api/cmd/tools/apidoc",
    ):
        if (ROOT / relative_path).exists():
            failures.append(f"{relative_path} must remain removed")

    schema_path = ROOT / "api/docs/route-catalog.schema.json"
    if not schema_path.exists():
        failures.append("api/docs/route-catalog.schema.json is missing")
    else:
        try:
            schema = json.loads(schema_path.read_text(encoding="utf-8"))
        except json.JSONDecodeError as error:
            failures.append(
                f"api/docs/route-catalog.schema.json is invalid JSON: {error}"
            )
        else:
            expected_required = {
                "kind",
                "schema_version",
                "active_starters",
                "routes",
            }
            if schema.get("$schema") != "https://json-schema.org/draft/2020-12/schema":
                failures.append(
                    "route catalog schema must use JSON Schema Draft 2020-12"
                )
            if schema.get("$id") != "urn:luas:schema:route-catalog:v1":
                failures.append("route catalog schema must use the stable version 1 URN")
            if schema.get("additionalProperties") is not False:
                failures.append("route catalog schema top-level object must be closed")
            if set(schema.get("required", [])) != expected_required:
                failures.append(
                    "route catalog schema must require exactly the version 1 fields"
                )

    require_executable(failures, "api/scripts/verify-route-catalog.sh")
    require_executable(failures, "api/scripts/validate-route-catalog.py")

    if failures:
        print("Route contract discovery check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print("Route contract discovery check passed (runtime assembly is canonical).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
