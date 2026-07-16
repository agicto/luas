#!/usr/bin/env python3

"""Keep production image identities, evidence, and scan policy executable."""

from __future__ import annotations

import datetime as dt
import re
import stat
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]
FRONTEND = (
    "# syntax=docker.io/docker/dockerfile:1.24.0@sha256:"
    "87999aa3d42bdc6bea60565083ee17e86d1f3339802f543c0d03998580f9cb89"
)
API_BUILDER = (
    "docker.io/library/golang:1.25.12-alpine3.24@sha256:"
    "56961d79ea8129efddcc0b8643fd8a5416b4e6228cfd477e3fd61deb2672c587"
)
API_RUNTIME = (
    "gcr.io/distroless/static-debian12:nonroot@sha256:"
    "aef9602f8710ec12bde19d593fed1f76c708531bb7aba205110f1029786ead7b"
)
WEB_BASE = (
    "docker.io/library/node:22.23.1-alpine3.24@sha256:"
    "16e22a550f3863206a3f701448c45f7912c6896a62de43add43bb9c86130c3e2"
)
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
IGNORE_ID_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._:-]*$")


def require_markers(failures: list[str], path: Path, markers: tuple[str, ...]) -> str:
    content = path.read_text(encoding="utf-8")
    relative = path.relative_to(ROOT)
    for marker in markers:
        if marker not in content:
            failures.append(f"{relative} must contain {marker!r}")
    return content


def check_external_from_pins(failures: list[str], path: Path, content: str) -> None:
    aliases: set[str] = set()
    for line_number, line in enumerate(content.splitlines(), start=1):
        match = re.match(
            r"^FROM\s+(?P<image>\S+)(?:\s+AS\s+(?P<alias>[A-Za-z0-9._-]+))?$",
            line,
            flags=re.IGNORECASE,
        )
        if match is None:
            continue
        image = match.group("image")
        if image != "scratch" and image not in aliases:
            reference, separator, digest = image.rpartition("@sha256:")
            image_name = reference.rsplit("/", maxsplit=1)[-1]
            if (
                not separator
                or not reference
                or ":" not in image_name
                or SHA256_PATTERN.fullmatch(digest) is None
            ):
                failures.append(
                    f"{path.relative_to(ROOT)}:{line_number} external FROM must use "
                    "an exact tag and sha256 digest"
                )
        alias = match.group("alias")
        if alias:
            aliases.add(alias)


def unquote(value: str) -> str:
    value = value.strip()
    if len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'":
        return value[1:-1]
    return value


def check_ignore_policy(failures: list[str]) -> None:
    path = ROOT / ".trivyignore.yaml"
    lines = path.read_text(encoding="utf-8").splitlines()
    entries: list[tuple[int, dict[str, str]]] = []
    current: tuple[int, dict[str, str]] | None = None

    for line_number, line in enumerate(lines, start=1):
        section = re.match(
            r"^(vulnerabilities|secrets|misconfigurations|licenses):\s*(?P<value>.*)$",
            line,
        )
        if section and section.group("value").strip() not in ("", "[]"):
            failures.append(
                f".trivyignore.yaml:{line_number} must not use compact exceptions; "
                "use structured id/statement/expired_at entries"
            )

    for line_number, line in enumerate(lines, start=1):
        entry = re.match(r"^(?P<indent>\s*)-\s+id:\s*(?P<value>[^#]+)", line)
        if entry:
            current = (
                len(entry.group("indent")),
                {"id": unquote(entry.group("value"))},
            )
            entries.append((line_number, current[1]))
            continue
        if current is None:
            continue
        indent, values = current
        field = re.match(
            r"^(?P<indent>\s+)(?P<key>statement|expired_at):\s*(?P<value>[^#]+)",
            line,
        )
        if field and len(field.group("indent")) > indent:
            values[field.group("key")] = unquote(field.group("value"))

    seen: set[str] = set()
    today = dt.date.today()
    for line_number, values in entries:
        identifier = values["id"]
        location = f".trivyignore.yaml:{line_number}"
        if IGNORE_ID_PATTERN.fullmatch(identifier) is None:
            failures.append(f"{location} must use one exact, non-wildcard finding ID")
        if identifier in seen:
            failures.append(f"{location} duplicates finding ID {identifier}")
        seen.add(identifier)
        if not values.get("statement", "").strip():
            failures.append(f"{location} must include a non-empty statement")
        expiry = values.get("expired_at", "")
        try:
            expiry_date = dt.date.fromisoformat(expiry)
        except ValueError:
            failures.append(f"{location} must include expired_at as YYYY-MM-DD")
        else:
            if expiry_date <= today:
                failures.append(f"{location} expired on {expiry_date.isoformat()}")


def main() -> int:
    failures: list[str] = []

    api_dockerfile = ROOT / "api/Dockerfile"
    api_content = require_markers(
        failures,
        api_dockerfile,
        (
            FRONTEND,
            f"FROM {API_BUILDER} AS builder",
            f"FROM {API_RUNTIME}",
            'org.opencontainers.image.title="Luas API"',
            "org.opencontainers.image.source",
            "org.opencontainers.image.revision",
            "org.opencontainers.image.version",
            "org.opencontainers.image.base.digest",
            'ENTRYPOINT ["/app/luas-server"]',
            "HEALTHCHECK",
        ),
    )
    if api_content.splitlines()[0] != FRONTEND:
        failures.append("api/Dockerfile must pin the reviewed frontend on line 1")
    check_external_from_pins(failures, api_dockerfile, api_content)

    web_dockerfile = ROOT / "web/Dockerfile"
    web_content = require_markers(
        failures,
        web_dockerfile,
        (
            FRONTEND,
            f"FROM {WEB_BASE} AS base",
            "FROM base AS runtime-root",
            "FROM scratch AS runner",
            "rm -rf /usr/local/lib/node_modules /usr/local/include",
            "/usr/local/share/doc /usr/local/share/man /opt/yarn-*",
            "/usr/local/bin/docker-entrypoint.sh /sbin/apk",
            "COPY --from=runtime-root / /",
            'org.opencontainers.image.title="Luas Web"',
            "org.opencontainers.image.source",
            "org.opencontainers.image.revision",
            "org.opencontainers.image.version",
            "org.opencontainers.image.base.digest",
            "USER 1000:1000",
            "HEALTHCHECK",
            'CMD ["node", "server.js"]',
        ),
    )
    if web_content.splitlines()[0] != FRONTEND:
        failures.append("web/Dockerfile must pin the reviewed frontend on line 1")
    check_external_from_pins(failures, web_dockerfile, web_content)

    for relative, expected_materials, expected_inputs in (
        (
            "api/scripts/verify-container.sh",
            3,
            (FRONTEND, API_BUILDER, API_RUNTIME),
        ),
        ("web/scripts/verify-container.sh", 2, (FRONTEND, WEB_BASE)),
    ):
        path = ROOT / relative
        require_markers(
            failures,
            path,
            (
                "docker buildx build",
                "BUILDX_METADATA_PROVENANCE=max",
                '--metadata-file "${BUILD_METADATA_OUTPUT}"',
                "https://mobyproject.org/buildkit@v1",
                f"len(materials) < {expected_materials}",
                "org.opencontainers.image.source",
                "org.opencontainers.image.revision",
                "org.opencontainers.image.version",
                "org.opencontainers.image.base.digest",
                "HEALTHCHECK",
            ),
        )
        script_content = path.read_text(encoding="utf-8")
        for expected_input in expected_inputs:
            digest = expected_input.rsplit("sha256:", maxsplit=1)[1]
            if digest not in script_content:
                failures.append(f"{relative} must verify material digest {digest}")
        if not (path.stat().st_mode & stat.S_IXUSR):
            failures.append(f"{relative} must be executable")

    scanner = ROOT / "scripts/container-security.sh"
    scanner_content = require_markers(
        failures,
        scanner,
        (
            'TRIVY_VERSION="0.72.0"',
            "88f208680dc05da2b459e19b4f5aa2b4dc7c2117892ba4aab2ae63baba330016",
            "ee5e60df8a98e5b89fd74a6d86f9e5c7e9a266a35002cb1e43291698b3bfee08",
            "bbb64b9695866ce4a7a8f5c9592002c5961cab378577fa3f8a040df362b9b2ea",
            "2ca2c023109c2db6b2b77366b6717291452d4531167377d95c79547f0c8e3467",
            "ed3cf122060f61818fe1f735fd97557954e16e10bc8b058af9852271cf2e91b3",
            "--detection-priority precise",
            "--scanners vuln,secret",
            "--severity HIGH,CRITICAL",
            "--exit-code 1",
            "--exit-on-eol 1",
            "CycloneDX 1.7",
            "checksum-verified archive every run",
            "validate-container-sbom.py",
        ),
    )
    if "--ignore-unfixed" in scanner_content:
        failures.append("scripts/container-security.sh must not hide unfixed findings")
    if not (scanner.stat().st_mode & stat.S_IXUSR):
        failures.append("scripts/container-security.sh must be executable")

    sbom_validator = ROOT / "scripts/validate-container-sbom.py"
    require_markers(
        failures,
        sbom_validator,
        (
            "aquasecurity:trivy:ImageID",
            "aquasecurity:trivy:Reference",
            "aquasecurity:trivy:RepoTag",
            "pkg:oci/",
            "--self-test",
        ),
    )
    if not (sbom_validator.stat().st_mode & stat.S_IXUSR):
        failures.append("scripts/validate-container-sbom.py must be executable")
    validator_test = subprocess.run(
        [sys.executable, str(sbom_validator), "--self-test"],
        check=False,
        capture_output=True,
        text=True,
    )
    if validator_test.returncode != 0:
        failures.append(
            "scripts/validate-container-sbom.py self-test failed: "
            f"{validator_test.stderr.strip() or validator_test.stdout.strip()}"
        )

    check_ignore_policy(failures)

    require_markers(
        failures,
        ROOT / ".github/workflows/container.yml",
        (
            "scripts/container-security.sh",
            "scripts/validate-container-sbom.py",
            "BUILD_METADATA_OUTPUT",
            "OCI_SOURCE: ${{ github.server_url }}/${{ github.repository }}",
            "luas-api.build-metadata.json",
            "luas-api.cdx.json",
            "luas-api-container-security-${{ github.sha }}",
            "retention-days: 14",
        ),
    )
    require_markers(
        failures,
        ROOT / ".github/workflows/web-container.yml",
        (
            "scripts/verify-container.sh luas-web:ci",
            "scripts/container-security.sh verify luas-web:ci",
            "scripts/validate-container-sbom.py",
            "OCI_SOURCE: ${{ github.server_url }}/${{ github.repository }}",
            "luas-web.build-metadata.json",
            "luas-web.cdx.json",
            "luas-web-container-security-${{ github.sha }}",
            "retention-days: 14",
        ),
    )
    require_markers(
        failures,
        ROOT / "Makefile",
        (
            "check-container-supply-chain.py",
            "container-scan:",
            "container-sbom:",
            'test -n "$${IMAGE:-}"',
        ),
    )
    require_markers(
        failures,
        ROOT / "docs/CONTAINER_SECURITY.md",
        (
            "Trivy 0.72.0",
            "CycloneDX 1.7",
            "HIGH/CRITICAL",
            "BuildKit",
            "14 days",
            "unsigned",
            "Cosign",
            "downstream",
            "IMAGE=luas-api:container-check",
        ),
    )

    if failures:
        print("Container supply-chain governance check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print(
        "Container supply-chain governance check passed "
        "(2 digest-pinned images, BuildKit evidence, CycloneDX 1.7, Trivy 0.72.0)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
