#!/usr/bin/env python3

"""Keep optional starter runtime, schema, contracts, and guidance aligned."""

from __future__ import annotations

import re
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


def between(content: str, start: str, end: str) -> str:
    if start not in content or end not in content:
        return ""
    return content.split(start, 1)[1].split(end, 1)[0]


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "api/internal/infra/config/config.go",
        (
            "type StarterConfig struct",
            'env.GetSlice("OPTIONAL_STARTERS", []string{})',
        ),
    )
    require_all(
        failures,
        "api/internal/starter/catalog.go",
        (
            "canonical lowercase name",
            "duplicate optional starter",
            "unknown optional starter",
            "is a default starter",
        ),
    )
    require_all(
        failures,
        "api/internal/starter/defaults.go",
        (
            "func NewConfiguredRegistry(",
            "func OptionalManifests(",
            "func ConfiguredMigrations(",
            "func ConfiguredSeeders(",
            "migrator.RegisterMany(registry.Migrations())",
        ),
    )
    require_all(
        failures,
        "api/internal/starter/assembly/starter_manifest.go",
        ("isNilModule", "reflect.ValueOf(module)"),
    )
    require_all(
        failures,
        "api/internal/starter/registry.go",
        (
            "migrationOwners",
            "seederOwners",
            "Activation happens only after every manifest contribution passes preflight.",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/console/commands/migrate.go",
        ("starter.ConfiguredMigrations(cfg)",),
    )
    require_all(
        failures,
        "api/internal/infra/console/commands/db.go",
        ("bootstrap.RunConfiguredSeeders(db, cfg)",),
    )

    try:
        defaults = read("api/internal/starter/defaults.go")
    except FileNotFoundError:
        defaults = ""
    default_segment = between(
        defaults, "func DefaultManifests(", "// DefaultMigrations"
    )
    optional_segment = between(
        defaults, "func OptionalManifests(", "// ConfiguredManifests"
    )
    optional_packages = re.findall(
        r"([a-z][a-z0-9_]*)\.NewStarterManifest", optional_segment
    )
    if optional_packages != ["organization"]:
        failures.append(
            "optional starter catalog must currently contain exactly organization"
        )
    if "organization.NewStarterManifest" in default_segment:
        failures.append("organization must not be part of DefaultManifests")

    for package_name in optional_packages:
        provider_path = f"api/internal/modules/{package_name}/provider.go"
        try:
            provider = read(provider_path)
        except FileNotFoundError:
            failures.append(f"{provider_path} is missing")
            continue
        if f'"{package_name}"' not in provider:
            failures.append(
                f"{provider_path} must use the canonical starter name {package_name!r}"
            )
        migration_names = re.findall(
            r'"([0-9]{4}_[0-9]{2}_[0-9]{2}_[0-9]{6}_[a-z0-9_]+)"',
            provider,
        )
        if not migration_names:
            failures.append(f"{provider_path} must register at least one migration")
        for migration_name in migration_names:
            migration_path = (
                ROOT / "api" / "database" / "migrations" / f"{migration_name}.go"
            )
            if not migration_path.exists():
                failures.append(
                    f"{provider_path} references missing migration {migration_name}"
                )

    require_all(
        failures,
        "contracts/ORGANIZATIONS.md",
        (
            "OPTIONAL_STARTERS=organization",
            "ORGANIZATION.NOT_FOUND",
            "ORGANIZATION.SLUG_ALREADY_EXISTS",
            "ORGANIZATION.OWNERSHIP_TRANSFER_REQUIRED",
            "Deliberate Deferrals",
        ),
    )
    require_all(
        failures,
        "api/internal/domain/error_codes.go",
        (
            '"ORGANIZATION.NOT_FOUND"',
            '"ORGANIZATION.SLUG_ALREADY_EXISTS"',
            '"ORGANIZATION.OWNERSHIP_TRANSFER_REQUIRED"',
        ),
    )
    require_all(
        failures,
        "api/database/migrations/2026_07_14_000000_create_organizations_tables.go",
        (
            "UseTransaction: true",
            "organization.OrganizationPO{}",
            "organization.OrganizationMembershipPO{}",
        ),
    )
    require_all(
        failures,
        "api/.env.example",
        ("OPTIONAL_STARTERS=", "available: organization"),
    )
    require_all(
        failures,
        "api/docker-compose.yml",
        ("OPTIONAL_STARTERS: ${OPTIONAL_STARTERS:-}",),
    )
    require_all(
        failures,
        "api/docs/CONFIGURATION.md",
        ("Optional Starter Activation", "OPTIONAL_STARTERS=organization"),
    )
    require_all(
        failures,
        "docs/STARTER_BUSINESS_ROADMAP.md",
        ("`organization` optional starter", "Foundation only"),
    )
    require_all(
        failures,
        "api/.agents/skills/module-creation/SKILL.md",
        ("OptionalManifests", "OPTIONAL_STARTERS=blog"),
    )

    if failures:
        print("Starter catalog check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print(
        f"Starter catalog check passed ({len(optional_packages)} optional starter)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
