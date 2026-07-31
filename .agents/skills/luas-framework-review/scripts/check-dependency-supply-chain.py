#!/usr/bin/env python3

"""Keep dependency resolution, scanning, and exception policy reproducible."""

from __future__ import annotations

import datetime as dt
import json
import re
import sys
import tomllib
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]
PNPM_VERSION = "10.34.5"
NODE_ENGINES = "^22.12.0 || ^24.0.0"
BROWSER_PROJECTS = ("web", "web-spa")
PNPM_PROJECTS = (*BROWSER_PROJECTS, "contracts")
OSV_VERSION = "2.3.8"
OSV_CHECKSUMS = {
    "osv-scanner_darwin_amd64": (
        "b8a80a9f14ca4c0cd0fc2d351b28f740da9e6a5b18385ac9f9d083360b5b504e"
    ),
    "osv-scanner_darwin_arm64": (
        "a8cd6507b06239f463a7642430cfd2d154882f150f6e30cdc0653e28dfc34216"
    ),
    "osv-scanner_linux_amd64": (
        "bc98e15319ed0d515e3f9235287ba53cdc5535d576d24fd573978ecfe9ab92dc"
    ),
    "osv-scanner_linux_arm64": (
        "8158b18edd2d03b1a30d905ca91b032bc62262167be8f206c27114f08823e27c"
    ),
    "osv-scanner_windows_amd64.exe": (
        "cb04e79dd9698a7bc821bbfdddec916a416d1409fda79c927c509d37d00c9716"
    ),
    "osv-scanner_windows_arm64.exe": (
        "285d1fbcf2c69ab5ee38ae3a850ab46e83f32ef1cd5f3c4c9eb161cc493f6d52"
    ),
}
ALLOWED_BUILDS = {
    "@parcel/watcher@2.5.1",
    "@swc/core@1.15.5",
    "esbuild@0.28.1",
    "sharp@0.34.5",
    "unrs-resolver@1.11.1",
}
DEPENDABOT_TARGETS = {
    ("npm", "/contracts"),
    ("npm", "/web"),
    ("npm", "/web-spa"),
    ("gomod", "/api"),
    ("github-actions", "/"),
    ("docker", "/api"),
    ("docker", "/web"),
}
DEPENDABOT_GROUPS = {
    "contracts-development-minor-patch",
    "web-production-minor-patch",
    "web-development-minor-patch",
    "web-spa-production-minor-patch",
    "web-spa-development-minor-patch",
    "go-minor-patch",
    "actions",
    "api-images",
    "web-images",
}


def top_level_scalar(content: str, key: str) -> str | None:
    match = re.search(rf"(?m)^{re.escape(key)}:\s*([^#\n]+?)\s*$", content)
    return match.group(1) if match else None


def parse_allow_builds(content: str) -> dict[str, bool]:
    lines = content.splitlines()
    try:
        start = lines.index("allowBuilds:") + 1
    except ValueError:
        return {}

    values: dict[str, bool] = {}
    for line in lines[start:]:
        if not line.startswith("  "):
            break
        match = re.fullmatch(r"  (.+): (true|false)", line)
        if match is None:
            continue
        key = match.group(1).strip("'\"")
        values[key] = match.group(2) == "true"
    return values


def parse_dependabot_targets(content: str) -> set[tuple[str, str]]:
    targets: set[tuple[str, str]] = set()
    entries = re.split(r"(?m)^  - package-ecosystem:\s*", content)[1:]
    for entry in entries:
        lines = entry.splitlines()
        ecosystem = lines[0].strip(" '\"")
        directory_match = re.search(r"(?m)^    directory:\s*([^#\n]+?)\s*$", entry)
        if directory_match:
            directory = directory_match.group(1).strip(" '\"")
            targets.add((ecosystem, directory))
    return targets


def parse_dependabot_groups(content: str) -> dict[str, str]:
    groups: dict[str, str] = {}
    pattern = re.compile(
        r"(?m)^      (?P<name>[a-z0-9-]+):\n"
        r"(?P<body>(?:        [^\n]*\n)+)"
    )
    for match in pattern.finditer(content):
        groups[match.group("name")] = match.group("body")
    return groups


def exception_date(value: object) -> dt.date | None:
    if isinstance(value, dt.date):
        return value
    if isinstance(value, str):
        try:
            return dt.date.fromisoformat(value)
        except ValueError:
            return None
    return None


def main() -> int:
    failures: list[str] = []

    expected_manager = f"pnpm@{PNPM_VERSION}"
    expected_scalars = {
        "packageManagerStrictVersion": "true",
        "minimumReleaseAge": "1440",
        "trustPolicy": "no-downgrade",
        "trustPolicyIgnoreAfter": "129600",
        "blockExoticSubdeps": "true",
        "strictDepBuilds": "true",
    }
    package_counts: dict[str, int] = {}

    for project in PNPM_PROJECTS:
        package_path = ROOT / project / "package.json"
        package = json.loads(package_path.read_text(encoding="utf-8"))
        if package.get("packageManager") != expected_manager:
            failures.append(
                f"{project}/package.json packageManager must be {expected_manager}"
            )
        if package.get("engines", {}).get("node") != NODE_ENGINES:
            failures.append(
                f"{project}/package.json engines.node must select supported LTS lines: "
                f"{NODE_ENGINES}"
            )
        if package.get("engines", {}).get("pnpm") != PNPM_VERSION:
            failures.append(
                f"{project}/package.json engines.pnpm must be {PNPM_VERSION}"
            )
        if project in BROWSER_PROJECTS:
            node_types = package.get("devDependencies", {}).get("@types/node", "")
            if not re.fullmatch(r"\^22\.\d+\.\d+", node_types):
                failures.append(
                    f"{project}/package.json @types/node must stay on the Node 22 baseline"
                )

        for lock_name in (
            "package-lock.json",
            "npm-shrinkwrap.json",
            "yarn.lock",
            "bun.lock",
            "bun.lockb",
        ):
            forbidden_lock = f"{project}/{lock_name}"
            if (ROOT / forbidden_lock).exists():
                failures.append(
                    f"{forbidden_lock} must not coexist with "
                    f"{project}/pnpm-lock.yaml"
                )

        workspace_path = ROOT / project / "pnpm-workspace.yaml"
        workspace = workspace_path.read_text(encoding="utf-8")
        for key, expected in expected_scalars.items():
            actual = top_level_scalar(workspace, key)
            if actual != expected:
                failures.append(
                    f"{project}/pnpm-workspace.yaml {key} must be {expected}, "
                    f"got {actual!r}"
                )

        builds = parse_allow_builds(workspace)
        if set(builds) != ALLOWED_BUILDS or not all(builds.values()):
            failures.append(
                f"{project}/pnpm-workspace.yaml allowBuilds must be the reviewed "
                "five-version allowlist"
            )
        for forbidden in (
            "onlyBuiltDependencies:",
            "ignoredBuiltDependencies:",
            "dangerouslyAllowAllBuilds:",
        ):
            if forbidden in workspace:
                failures.append(
                    f"{project}/pnpm-workspace.yaml must not use {forbidden}"
                )

        lock_path = ROOT / project / "pnpm-lock.yaml"
        lockfile = lock_path.read_text(encoding="utf-8")
        if "lockfileVersion: '9.0'" not in lockfile:
            failures.append(
                f"{project}/pnpm-lock.yaml must use the reviewed v9 lockfile format"
            )
        packages_match = re.search(
            r"(?ms)^packages:\n(?P<body>.*?)^snapshots:\n", lockfile
        )
        package_count = 0
        integrity_count = 0
        if packages_match is None:
            failures.append(
                f"{project}/pnpm-lock.yaml must contain packages and snapshots sections"
            )
        else:
            package_body = packages_match.group("body")
            package_count = len(re.findall(r"(?m)^  \S.*:\s*$", package_body))
            integrity_count = len(
                re.findall(r"(?m)^    resolution: \{integrity:", package_body)
            )
            if package_count == 0 or integrity_count != package_count:
                failures.append(
                    f"every {project} lockfile package must carry registry integrity "
                    f"evidence ({integrity_count}/{package_count})"
                )
            for forbidden_source in ("git+", "http://", "tarball:"):
                if forbidden_source in package_body:
                    failures.append(
                        f"{project}/pnpm-lock.yaml contains forbidden source "
                        f"{forbidden_source!r}"
                    )
        package_counts[project] = package_count

    scanner_path = ROOT / "scripts/dependency-security.sh"
    scanner = scanner_path.read_text(encoding="utf-8")
    required_scanner_markers = (
        f'OSV_VERSION="{OSV_VERSION}"',
        "--proto '=https' --proto-redir '=https' --tlsv1.2",
        '"--config=${ROOT_DIR}/osv-scanner.toml"',
        '"--lockfile=${ROOT_DIR}/api/go.mod"',
        '"--lockfile=${ROOT_DIR}/contracts/pnpm-lock.yaml"',
        '"--lockfile=${ROOT_DIR}/web/pnpm-lock.yaml"',
        '"--lockfile=${ROOT_DIR}/web-spa/pnpm-lock.yaml"',
        "--format=cyclonedx-1-5",
        'document.get("bomFormat") != "CycloneDX"',
        'document.get("specVersion") != "1.5"',
    )
    for marker in required_scanner_markers:
        if marker not in scanner:
            failures.append(f"scripts/dependency-security.sh must contain {marker!r}")
    for asset, checksum in OSV_CHECKSUMS.items():
        if asset not in scanner or checksum not in scanner:
            failures.append(f"scripts/dependency-security.sh must pin {asset} by SHA-256")

    config_path = ROOT / "osv-scanner.toml"
    with config_path.open("rb") as handle:
        config = tomllib.load(handle)
    today = dt.datetime.now(dt.timezone.utc).date()
    ignored = config.get("IgnoredVulns", [])
    overrides = config.get("PackageOverrides", [])
    for index, item in enumerate(ignored, start=1):
        if not str(item.get("id", "")).strip():
            failures.append(f"IgnoredVulns entry {index} must include id")
        if not str(item.get("reason", "")).strip():
            failures.append(f"IgnoredVulns entry {index} must include reason")
        expiry = exception_date(item.get("ignoreUntil"))
        if expiry is None or expiry < today:
            failures.append(
                f"IgnoredVulns entry {index} must include a non-expired ignoreUntil date"
            )
    for index, item in enumerate(overrides, start=1):
        if not str(item.get("reason", "")).strip():
            failures.append(f"PackageOverrides entry {index} must include reason")
        expiry = exception_date(item.get("effectiveUntil"))
        if expiry is None or expiry < today:
            failures.append(
                f"PackageOverrides entry {index} must include a non-expired effectiveUntil date"
            )

    dependabot_path = ROOT / ".github/dependabot.yml"
    dependabot = dependabot_path.read_text(encoding="utf-8")
    if not dependabot.startswith("version: 2\n"):
        failures.append(".github/dependabot.yml must use version 2")
    actual_targets = parse_dependabot_targets(dependabot)
    if actual_targets != DEPENDABOT_TARGETS:
        failures.append(
            ".github/dependabot.yml targets must be npm/contracts+web+web-spa, gomod/api, "
            "GitHub Actions/root, and Docker/api+web"
        )
    if dependabot.count("interval: weekly") != len(DEPENDABOT_TARGETS):
        failures.append("every Dependabot target must run weekly")
    groups = parse_dependabot_groups(dependabot)
    if set(groups) != DEPENDABOT_GROUPS:
        failures.append("Dependabot must keep the nine reviewed minor/patch update groups")
    for name, body in groups.items():
        if "update-types: [minor, patch]" not in body:
            failures.append(
                f"Dependabot group {name!r} must keep major updates as separate reviews"
            )

    workflow = (ROOT / ".github/workflows/dependency-security.yml").read_text(
        encoding="utf-8"
    )
    for marker in (
        'cron: "17 5 * * 1"',
        "bash scripts/dependency-security.sh sbom",
        "bash scripts/dependency-security.sh scan",
        "actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1",
        "retention-days: 14",
    ):
        if marker not in workflow:
            failures.append(
                f".github/workflows/dependency-security.yml must contain {marker!r}"
            )

    makefile = (ROOT / "Makefile").read_text(encoding="utf-8")
    for marker in (
        "check-dependency-supply-chain.py",
        "dependency-scan:",
        "scripts/dependency-security.sh scan",
        "sbom:",
        "scripts/dependency-security.sh sbom",
    ):
        if marker not in makefile:
            failures.append(f"Makefile must contain {marker!r}")

    ci_doc = (ROOT / "docs/CI.md").read_text(encoding="utf-8")
    dependency_doc = (ROOT / "docs/DEPENDENCY_SECURITY.md").read_text(
        encoding="utf-8"
    )
    for marker in (
        "dependency-security.yml",
        "CycloneDX 1.5",
        "OSV-Scanner 2.3.8",
        "make dependency-scan",
    ):
        if marker not in ci_doc + dependency_doc:
            failures.append(f"dependency governance docs must contain {marker!r}")

    if failures:
        print("Dependency supply-chain check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print(
        "Dependency supply-chain check passed "
        f"(Node 22/24 LTS, pnpm {PNPM_VERSION}, "
        f"{sum(package_counts.values())} integrity-pinned pnpm packages, "
        f"OSV-Scanner {OSV_VERSION}, {len(ignored) + len(overrides)} exceptions)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
