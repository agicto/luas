#!/usr/bin/env python3

"""Enforce reviewed, immutable GitHub Action dependencies and CI boundaries."""

from __future__ import annotations

import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]
WORKFLOW_ROOT = ROOT / ".github/workflows"
USES_PATTERN = re.compile(
    r"^\s*uses:\s*(?P<action>[^@\s#]+)@(?P<ref>[^\s#]+)"
    r"(?:\s+#\s*(?P<version>\S+))?\s*$"
)
FULL_SHA_PATTERN = re.compile(r"^[0-9a-f]{40}$")
PERMISSION_PATTERN = re.compile(r"^\s+(?P<scope>[a-z-]+):\s*(?P<access>read|write|none)\s*$")
PRIVILEGED_PR_TRIGGER_PATTERN = re.compile(
    r'''(?m)^(?:pull_request_target|"pull_request_target"|'pull_request_target')\s*:'''
)


@dataclass(frozen=True)
class ActionPin:
    sha: str
    version: str
    runtime: str


REVIEWED_ACTIONS = {
    "actions/checkout": ActionPin(
        sha="93cb6efe18208431cddfb8368fd83d5badbf9bfd",
        version="v5.0.1",
        runtime="node24",
    ),
    "actions/setup-go": ActionPin(
        sha="4a3601121dd01d1626a1e23e37211e3254c1c06c",
        version="v6.4.0",
        runtime="node24",
    ),
    "actions/setup-node": ActionPin(
        sha="48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e",
        version="v6.4.0",
        runtime="node24",
    ),
    "actions/upload-artifact": ActionPin(
        sha="043fb46d1a93c77aae656e7c1c64a875d1fc6a0a",
        version="v7.0.1",
        runtime="node24",
    ),
    "golangci/golangci-lint-action": ActionPin(
        sha="ba0d7d2ec06a0ea1cb5fa41b2e4a3ab91d21278a",
        version="v9.3.0",
        runtime="node24",
    ),
    "pnpm/action-setup": ActionPin(
        sha="0ebf47130e4866e96fce0953f49152a61190b271",
        version="v6.0.9",
        runtime="node24",
    ),
}

REVIEWED_WORKFLOW_PERMISSIONS = {
    "ci.yml": {"contents": "read"},
    "container.yml": {"contents": "read"},
    "dependency-security.yml": {"contents": "read"},
    "skill-self-test.yml": {"contents": "read"},
    "sync-deploy-branches.yml": {"contents": "write"},
    "web-container.yml": {"contents": "read"},
}


def workflow_paths() -> list[Path]:
    return sorted((*WORKFLOW_ROOT.glob("*.yml"), *WORKFLOW_ROOT.glob("*.yaml")))


def action_repository(action: str) -> str:
    parts = action.split("/")
    if len(parts) < 2:
        return action
    return "/".join(parts[:2])


def top_level_permissions(content: str) -> dict[str, str] | None:
    lines = content.splitlines()
    for index, line in enumerate(lines):
        if line != "permissions:":
            continue

        permissions: dict[str, str] = {}
        for permission_line in lines[index + 1 :]:
            if not permission_line.strip() or permission_line.lstrip().startswith("#"):
                continue
            if not permission_line.startswith((" ", "\t")):
                break

            match = PERMISSION_PATTERN.fullmatch(permission_line)
            if match is None:
                return None
            permissions[match.group("scope")] = match.group("access")
        return permissions

    return None


def main() -> int:
    failures: list[str] = []
    occurrences: dict[str, int] = {name: 0 for name in REVIEWED_ACTIONS}
    workflows = workflow_paths()

    if not workflows:
        failures.append(".github/workflows contains no workflow files")

    for path in workflows:
        relative = path.relative_to(ROOT)
        content = path.read_text(encoding="utf-8")
        expected_permissions = REVIEWED_WORKFLOW_PERMISSIONS.get(path.name)
        actual_permissions = top_level_permissions(content)
        if expected_permissions is None:
            failures.append(f"{relative} has no reviewed token-permission policy")
        elif actual_permissions != expected_permissions:
            failures.append(
                f"{relative} permissions must be {expected_permissions}, "
                f"got {actual_permissions}"
            )
        if PRIVILEGED_PR_TRIGGER_PATTERN.search(content):
            failures.append(f"{relative} must not use pull_request_target")

        for line_number, line in enumerate(content.splitlines(), start=1):
            if "uses:" not in line:
                continue
            value = line.split("uses:", 1)[1].strip()
            if value.startswith("./"):
                continue

            match = USES_PATTERN.match(line)
            if match is None:
                failures.append(
                    f"{relative}:{line_number} has an unparseable external action reference"
                )
                continue

            action = match.group("action")
            reference = match.group("ref")
            version = match.group("version")
            repository = action_repository(action)
            reviewed = REVIEWED_ACTIONS.get(repository)

            if not FULL_SHA_PATTERN.fullmatch(reference):
                failures.append(
                    f"{relative}:{line_number} must pin {action} to a full commit SHA"
                )
                continue
            if reviewed is None:
                failures.append(
                    f"{relative}:{line_number} uses unreviewed action repository {repository}"
                )
                continue

            occurrences[repository] += 1
            if reference != reviewed.sha:
                failures.append(
                    f"{relative}:{line_number} uses unreviewed SHA for {repository}"
                )
            if version != reviewed.version:
                failures.append(
                    f"{relative}:{line_number} must annotate {repository} with "
                    f"# {reviewed.version}"
                )
            if reviewed.runtime != "node24":
                failures.append(f"{repository} is not reviewed for the Node 24 action runtime")

    for repository, count in occurrences.items():
        if count == 0:
            failures.append(f"reviewed action {repository} is not used by any workflow")

    missing_workflows = set(REVIEWED_WORKFLOW_PERMISSIONS).difference(
        path.name for path in workflows
    )
    for workflow in sorted(missing_workflows):
        failures.append(f"reviewed workflow .github/workflows/{workflow} is missing")

    ci_workflow = (WORKFLOW_ROOT / "ci.yml").read_text(encoding="utf-8")
    if "package_json_file: web/package.json" not in ci_workflow:
        failures.append("ci.yml must source the pnpm version from web/package.json")
    for marker in (
        'node-version: ["22", "24"]',
        "node-version: ${{ matrix.node-version }}",
        "fail-fast: false",
    ):
        if marker not in ci_workflow:
            failures.append(f"ci.yml must keep the Web Node 22/24 matrix marker {marker!r}")
    package_data = json.loads((ROOT / "web/package.json").read_text(encoding="utf-8"))
    package_manager = package_data.get("packageManager", "")
    if not re.fullmatch(r"pnpm@\d+\.\d+\.\d+", package_manager):
        failures.append("web/package.json must pin an exact pnpm packageManager version")

    container_workflow = (WORKFLOW_ROOT / "container.yml").read_text(encoding="utf-8")
    container_check_at = container_workflow.find(
        "bash scripts/verify-container.sh luas-api:ci"
    )
    compose_check_at = container_workflow.find(
        "bash scripts/verify-compose.sh luas-api:ci"
    )
    if (
        container_check_at < 0
        or compose_check_at < 0
        or container_check_at > compose_check_at
    ):
        failures.append(
            "container.yml must verify luas-api:ci before reusing it for Compose"
        )

    web_container_workflow = (WORKFLOW_ROOT / "web-container.yml").read_text(
        encoding="utf-8"
    )
    for marker in (
        "bash scripts/verify-container.sh luas-web:ci",
        "bash ../scripts/container-security.sh verify luas-web:ci",
        "luas-web-container-security-${{ github.sha }}",
    ):
        if marker not in web_container_workflow:
            failures.append(f"web-container.yml must contain {marker!r}")

    compose_verifier = (ROOT / "api/scripts/verify-compose.sh").read_text(
        encoding="utf-8"
    )
    for marker in (
        "if (( $# == 0 )); then",
        'docker build --progress=plain --tag "${IMAGE_TAG}" "${ROOT_DIR}"',
        "explicit image ${IMAGE_TAG} does not exist",
    ):
        if marker not in compose_verifier:
            failures.append(
                f"api/scripts/verify-compose.sh must contain {marker!r}"
            )

    ci_doc = (ROOT / "docs/CI.md").read_text(encoding="utf-8")
    for marker in (
        "full-length commit SHA",
        "Node 24",
        "v2.327.1",
        "packageManager",
        "pull_request_target",
        "false green",
    ):
        if marker not in ci_doc:
            failures.append(f"docs/CI.md must contain {marker!r}")

    if failures:
        print("CI action governance check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    total = sum(occurrences.values())
    print(
        "CI action governance check passed "
        f"({len(workflows)} workflows, {total} immutable action references, Node 24)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
