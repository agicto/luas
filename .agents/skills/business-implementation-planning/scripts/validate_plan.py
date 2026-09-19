#!/usr/bin/env python3
"""Mechanical lint for Luas business implementation plans.

This intentionally checks structure and obvious placeholders only. It cannot
decide whether product rules, ownership, contracts, or recovery are correct.
"""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path


READINESS = re.compile(
    r"^\s*readiness\s*:\s*(READY WITH RECORDED ASSUMPTIONS|READY|BLOCKED)\s*$",
    re.IGNORECASE | re.MULTILINE,
)
MUTATION_ROUTE = re.compile(
    r"(?:\|\s*|`)(POST|PUT|PATCH|DELETE)(?:\s*\||\s+)", re.IGNORECASE
)
COMMAND = re.compile(r"^\s*COMMAND\s+[A-Za-z][A-Za-z0-9_]*\s*\(", re.MULTILINE)
PLACEHOLDERS = (
    (re.compile(r"\b(?:TBD|TODO|FIXME|XXX)\b", re.IGNORECASE), "unfinished token"),
    (re.compile(r"\?\?\?"), "question-mark placeholder"),
    (re.compile(r"<[A-Z][A-Z0-9_ ./:-]{1,64}>"), "template placeholder"),
    (
        re.compile(
            r"\b(?:same as above|complete object)\b|\u540c\u4e0a|\u5b8c\u6574\u5bf9\u8c61",
            re.IGNORECASE,
        ),
        "implicit contract placeholder",
    ),
)
SECTIONS = {
    "scope": ("executive summary", "scope", "\u6982\u8ff0", "\u8303\u56f4"),
    "actors": ("actors", "access", "\u89d2\u8272", "\u53c2\u4e0e\u8005", "\u6743\u9650"),
    "workflow": ("workflow", "\u6d41\u7a0b"),
    "modules": ("module", "\u6a21\u5757"),
    "domain/data": ("domain", "persistence", "\u9886\u57df", "\u6570\u636e", "\u6301\u4e45\u5316"),
    "state": ("state machine", "\u72b6\u6001\u673a"),
    "contract": ("http contract", "api contract", "\u63a5\u53e3", "\u5951\u7ea6"),
    "async": ("asynchronous", "event", "\u5f02\u6b65", "\u4e8b\u4ef6"),
    "security/operations": ("security", "operations", "\u5b89\u5168", "\u8fd0\u7ef4", "\u53ef\u89c2\u6d4b"),
    "verification": ("test", "verification", "\u6d4b\u8bd5", "\u9a8c\u8bc1"),
    "delivery": ("delivery", "\u5b9e\u65bd", "\u4ea4\u4ed8", "\u9636\u6bb5"),
    "acceptance": ("acceptance", "\u9a8c\u6536"),
    "readiness": ("readiness", "\u5c31\u7eea"),
}


@dataclass(frozen=True)
class Finding:
    level: str
    message: str


def markdown_headings(text: str) -> list[str]:
    return [match.group(1).strip().lower() for match in re.finditer(r"^#{1,6}\s+(.+)$", text, re.MULTILINE)]


def fences_balanced(text: str) -> bool:
    stack: list[tuple[str, int]] = []
    for line in text.splitlines():
        match = re.match(r"^\s*(`{3,}|~{3,})", line)
        if not match:
            continue
        marker = match.group(1)
        kind = marker[0]
        if stack and stack[-1][0] == kind and len(marker) >= stack[-1][1]:
            stack.pop()
        elif not stack:
            stack.append((kind, len(marker)))
    return not stack


def open_decisions_resolved(text: str) -> bool:
    match = re.search(r"^(#{1,6})\s+.*open decisions.*$", text, re.IGNORECASE | re.MULTILINE)
    if not match:
        return True
    level = len(match.group(1))
    rest = text[match.end() :]
    next_heading = re.search(rf"^#{{1,{level}}}\s+", rest, re.MULTILINE)
    body = rest[: next_heading.start()] if next_heading else rest
    return bool(
        re.search(
            r"\b(?:none|not applicable)\b|\u65e0\u963b\u585e|\u6ca1\u6709?\u5f85\u51b3\u7b56",
            body,
            re.IGNORECASE,
        )
    )


def validate(text: str) -> list[Finding]:
    findings: list[Finding] = []
    if not fences_balanced(text):
        findings.append(Finding("error", "Markdown code fences are unbalanced"))

    for pattern, label in PLACEHOLDERS:
        match = pattern.search(text)
        if match:
            line = text.count("\n", 0, match.start()) + 1
            findings.append(Finding("error", f"{label} at line {line}: {match.group(0)!r}"))

    readiness = READINESS.findall(text)
    if len(readiness) != 1:
        findings.append(Finding("error", f"expected exactly one readiness result, found {len(readiness)}"))
    elif readiness[0].upper() != "BLOCKED" and not open_decisions_resolved(text):
        findings.append(Finding("error", "non-blocked plan has unresolved Open Decisions"))

    headings = markdown_headings(text)
    for name, aliases in SECTIONS.items():
        if not any(any(alias in heading for alias in aliases) for heading in headings):
            findings.append(Finding("warning", f"missing recognizable {name} section"))

    mutation_count = len(MUTATION_ROUTE.findall(text))
    command_count = len(COMMAND.findall(text))
    if mutation_count and command_count < mutation_count:
        findings.append(
            Finding(
                "warning",
                f"found {mutation_count} mutation routes but only {command_count} COMMAND pseudocode blocks",
            )
        )

    lowered = text.lower()
    required_mechanics = {
        "stable error_code": "error_code",
        "transaction boundary": "begin transaction",
        "response behavior": "return ",
    }
    for label, token in required_mechanics.items():
        if token not in lowered:
            findings.append(Finding("warning", f"missing {label}"))
    return findings


def run_self_test() -> int:
    valid = """# Plan
## Executive Summary
## Actors and Access
## Workflow
## Module
## Domain and Persistence
## State Machine
## HTTP Contract
| POST | `/v1/items` |
```text
COMMAND CreateItem(actor)
BEGIN TRANSACTION item
COMMIT
RETURN 201 ItemResponse
```
error_code
## Events and Asynchronous Work
## Security and Operations
## Test and Verification
## Delivery
## Acceptance
## Readiness
readiness: READY
"""
    invalid = valid.replace("readiness: READY", "readiness: READY\nTBD")
    localized = (
        valid.replace("Executive Summary", "\u6982\u8ff0")
        .replace("Actors and Access", "\u89d2\u8272\u4e0e\u6743\u9650")
        .replace("Workflow", "\u6d41\u7a0b")
        .replace("Module", "\u6a21\u5757")
        .replace("Domain and Persistence", "\u9886\u57df\u4e0e\u6301\u4e45\u5316")
        .replace("State Machine", "\u72b6\u6001\u673a")
        .replace("HTTP Contract", "\u63a5\u53e3\u5951\u7ea6")
        .replace("Events and Asynchronous Work", "\u4e8b\u4ef6\u4e0e\u5f02\u6b65\u4efb\u52a1")
        .replace("Security and Operations", "\u5b89\u5168\u4e0e\u8fd0\u7ef4")
        .replace("Test and Verification", "\u6d4b\u8bd5\u4e0e\u9a8c\u8bc1")
        .replace("Delivery", "\u4ea4\u4ed8\u9636\u6bb5")
        .replace("Acceptance", "\u9a8c\u6536")
        .replace("Readiness", "\u5c31\u7eea")
    )
    if (
        validate(valid)
        or validate(localized)
        or not any(item.level == "error" for item in validate(invalid))
    ):
        print("self-test failed", file=sys.stderr)
        return 1
    print("self-test passed")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("plans", nargs="*", type=Path)
    parser.add_argument("--strict", action="store_true", help="treat warnings as failures")
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args()
    if args.self_test:
        return run_self_test()
    if not args.plans:
        parser.error("provide at least one plan path")

    failed = False
    for path in args.plans:
        try:
            text = path.read_text(encoding="utf-8")
        except (OSError, UnicodeError) as error:
            print(f"{path}: error: {error}")
            failed = True
            continue
        findings = validate(text)
        for finding in findings:
            print(f"{path}: {finding.level}: {finding.message}")
        errors = sum(item.level == "error" for item in findings)
        warnings = sum(item.level == "warning" for item in findings)
        print(f"{path}: {errors} error(s), {warnings} warning(s)")
        failed = failed or errors > 0 or (args.strict and warnings > 0)
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
