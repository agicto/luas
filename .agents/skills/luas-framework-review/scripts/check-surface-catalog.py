#!/usr/bin/env python3

"""Check Luas scaffold surface classifications across docs and skills."""

from __future__ import annotations

import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]
SURFACE_DOC = ROOT / "docs" / "SCAFFOLD_SURFACES.md"
DOWNSTREAM_SKILL = ROOT / ".agents" / "skills" / "downstream-app-extraction" / "SKILL.md"
CONTEXT = ROOT / "CONTEXT.md"

REQUIRED_SURFACES = {
    "core",
    "default starter",
    "optional starter",
    "capability",
    "mock bff",
    "console",
    "devtools",
    "example",
    "product-specific behavior",
}
CONTEXT_TERMS = {
    "Core": "core",
    "Default starter": "default starter",
    "Optional starter": "optional starter",
    "Capability": "capability",
    "Mock BFF": "mock bff",
    "Console": "console",
    "Devtools": "devtools",
    "Example": "example",
}
REQUIRED_REFERENCES = [
    "../CONTEXT.md",
    "../contracts/README.md",
    "../web/docs/MOCK_BFF.md",
    "../api/docs/ADDING_MODULE.md",
    "../web/docs/ADDING_FEATURE.md",
    "../admin/docs/ADDING_FEATURE.md",
    "check-downstream-contamination.sh",
    "check-surface-catalog.py",
    "check-starter-catalog.py",
    "check-permission-boundary.py",
    "check-setting-boundary.py",
    "check-usage-boundary.py",
    "check-webhook-boundary.py",
]
TABLE_ROW_RE = re.compile(r"^\|(.+)\|$")


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def normalize_surface(cell: str) -> str:
    return cell.strip().strip("`").lower()


def table_rows(path: Path) -> dict[str, list[str]]:
    rows: dict[str, list[str]] = {}

    for line in read(path).splitlines():
        match = TABLE_ROW_RE.match(line)
        if not match:
            continue

        cells = [cell.strip() for cell in line.strip().strip("|").split("|")]
        if not cells:
            continue

        surface = normalize_surface(cells[0])
        if surface in REQUIRED_SURFACES:
            rows[surface] = cells

    return rows


def missing_surfaces(label: str, rows: dict[str, list[str]]) -> list[str]:
    missing = sorted(REQUIRED_SURFACES - set(rows))
    if not missing:
        return []
    return [f"{label} is missing surface row(s): {', '.join(missing)}"]


def unexpected_surfaces(label: str, rows: dict[str, list[str]]) -> list[str]:
    unexpected = sorted(set(rows) - REQUIRED_SURFACES)
    if not unexpected:
        return []
    return [f"{label} has unexpected surface row(s): {', '.join(unexpected)}"]


def check_catalog_rows(rows: dict[str, list[str]]) -> list[str]:
    failures: list[str] = []

    for surface, cells in sorted(rows.items()):
        if len(cells) < 5:
            failures.append(f"docs/SCAFFOLD_SURFACES.md row for {surface} must include action and verification columns")
            continue

        if not cells[3] or not cells[4]:
            failures.append(f"docs/SCAFFOLD_SURFACES.md row for {surface} must include downstream action and verification")

    return failures


def main() -> int:
    failures: list[str] = []

    if not SURFACE_DOC.exists():
        failures.append("docs/SCAFFOLD_SURFACES.md is missing")
    if not DOWNSTREAM_SKILL.exists():
        failures.append(".agents/skills/downstream-app-extraction/SKILL.md is missing")
    if failures:
        return fail(failures)

    catalog_rows = table_rows(SURFACE_DOC)
    skill_rows = table_rows(DOWNSTREAM_SKILL)
    context_text = read(CONTEXT)
    surface_text = read(SURFACE_DOC)

    failures.extend(missing_surfaces("docs/SCAFFOLD_SURFACES.md", catalog_rows))
    failures.extend(unexpected_surfaces("docs/SCAFFOLD_SURFACES.md", catalog_rows))
    failures.extend(missing_surfaces("downstream-app-extraction skill", skill_rows))
    failures.extend(check_catalog_rows(catalog_rows))

    for title, normalized in CONTEXT_TERMS.items():
        if f"**{title}**" not in context_text:
            failures.append(f"CONTEXT.md is missing glossary term for {normalized}")

    for reference in REQUIRED_REFERENCES:
        if reference not in surface_text:
            failures.append(f"docs/SCAFFOLD_SURFACES.md must reference {reference}")

    if failures:
        return fail(failures)

    print(f"Scaffold surface catalog check passed ({len(REQUIRED_SURFACES)} surfaces).")
    return 0


def fail(failures: list[str]) -> int:
    print("Scaffold surface catalog check failed:", file=sys.stderr)
    for failure in failures:
        print(f"  {failure}", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
