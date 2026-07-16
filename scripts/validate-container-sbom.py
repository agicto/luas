#!/usr/bin/env python3

"""Validate Trivy CycloneDX image identity across Docker storage backends."""

from __future__ import annotations

import copy
import json
import re
import sys
from pathlib import Path
from typing import Any


SHA256_PATTERN = re.compile(r"^sha256:[0-9a-f]{64}$")


def property_values(subject: dict[str, Any], name: str) -> list[str]:
    properties = subject.get("properties")
    if not isinstance(properties, list):
        return []
    return [
        value
        for prop in properties
        if isinstance(prop, dict) and prop.get("name") == name
        for value in [prop.get("value")]
        if isinstance(value, str)
    ]


def validate_document(document: Any, expected_image: str) -> tuple[int, int]:
    if not isinstance(document, dict):
        raise ValueError("container SBOM must be a JSON object")
    if document.get("bomFormat") != "CycloneDX" or document.get("specVersion") != "1.7":
        raise ValueError("Trivy did not produce a CycloneDX 1.7 document")

    components = document.get("components")
    if not isinstance(components, list) or not components:
        raise ValueError("container CycloneDX document contains no components")

    vulnerabilities = document.get("vulnerabilities", [])
    if not isinstance(vulnerabilities, list):
        raise ValueError("container CycloneDX vulnerabilities must be a list")

    metadata = document.get("metadata")
    if not isinstance(metadata, dict):
        raise ValueError("container CycloneDX document is missing metadata")
    subject = metadata.get("component", {})
    if not isinstance(subject, dict) or subject.get("type") != "container":
        raise ValueError(
            "container CycloneDX metadata is missing its container subject"
        )
    if not isinstance(subject.get("name"), str) or not subject["name"]:
        raise ValueError("container CycloneDX subject is missing its display name")
    if not isinstance(subject.get("bom-ref"), str) or not subject["bom-ref"]:
        raise ValueError("container CycloneDX subject is missing bom-ref identity")

    purl = subject.get("purl")
    if purl is not None and (
        not isinstance(purl, str) or not purl.startswith("pkg:oci/")
    ):
        raise ValueError("container CycloneDX subject has an invalid OCI purl")

    image_ids = property_values(subject, "aquasecurity:trivy:ImageID")
    if not image_ids or not all(SHA256_PATTERN.fullmatch(value) for value in image_ids):
        raise ValueError("container CycloneDX subject is missing a valid Trivy ImageID")

    references = {
        *property_values(subject, "aquasecurity:trivy:Reference"),
        *property_values(subject, "aquasecurity:trivy:RepoTag"),
    }
    if expected_image not in references:
        raise ValueError(
            "container CycloneDX subject is missing the requested image reference"
        )

    return len(components), len(vulnerabilities)


def fixture_subject(image: str, *, with_purl: bool) -> dict[str, Any]:
    subject: dict[str, Any] = {
        "type": "container",
        "name": image,
        "bom-ref": "portable-local-image-reference",
        "properties": [
            {
                "name": "aquasecurity:trivy:ImageID",
                "value": f"sha256:{'a' * 64}",
            },
            {"name": "aquasecurity:trivy:Reference", "value": image},
            {"name": "aquasecurity:trivy:RepoTag", "value": image},
        ],
    }
    if with_purl:
        subject["purl"] = f"pkg:oci/{image.split(':', maxsplit=1)[0]}@sha256:{'b' * 64}"
        subject["bom-ref"] = subject["purl"]
    return subject


def run_self_test() -> int:
    image = "luas-web:ci"
    base = {
        "bomFormat": "CycloneDX",
        "specVersion": "1.7",
        "metadata": {"component": fixture_subject(image, with_purl=False)},
        "components": [{"type": "library", "name": "next", "version": "16.2.9"}],
        "vulnerabilities": [],
    }
    validate_document(base, image)

    registry_form = copy.deepcopy(base)
    registry_form["metadata"]["component"] = fixture_subject(image, with_purl=True)
    validate_document(registry_form, image)

    missing_id = copy.deepcopy(base)
    missing_id["metadata"]["component"]["properties"] = [
        prop
        for prop in missing_id["metadata"]["component"]["properties"]
        if prop["name"] != "aquasecurity:trivy:ImageID"
    ]
    try:
        validate_document(missing_id, image)
    except ValueError as error:
        if "ImageID" not in str(error):
            raise
    else:
        raise AssertionError("missing ImageID fixture unexpectedly passed")

    wrong_reference = copy.deepcopy(base)
    wrong_reference["metadata"]["component"]["properties"] = [
        {**prop, "value": "other:ci"}
        if prop["name"]
        in {
            "aquasecurity:trivy:Reference",
            "aquasecurity:trivy:RepoTag",
        }
        else prop
        for prop in wrong_reference["metadata"]["component"]["properties"]
    ]
    try:
        validate_document(wrong_reference, image)
    except ValueError as error:
        if "requested image reference" not in str(error):
            raise
    else:
        raise AssertionError("wrong image reference fixture unexpectedly passed")

    print("Container SBOM validator self-test passed (daemon and OCI purl forms).")
    return 0


def main() -> int:
    if sys.argv[1:] == ["--self-test"]:
        return run_self_test()
    if len(sys.argv) != 3:
        print(
            "usage: validate-container-sbom.py <sbom.cdx.json> <expected-image>",
            file=sys.stderr,
        )
        return 2

    path = Path(sys.argv[1])
    expected_image = sys.argv[2]
    try:
        with path.open(encoding="utf-8") as handle:
            document = json.load(handle)
        components, vulnerabilities = validate_document(document, expected_image)
    except (OSError, json.JSONDecodeError, ValueError) as error:
        print(f"container SBOM validation failed: {error}", file=sys.stderr)
        return 1

    print(
        f"Validated CycloneDX 1.7 image SBOM with {components} components "
        f"and {vulnerabilities} recorded vulnerabilities."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
