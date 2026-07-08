#!/usr/bin/env python3

"""Check local Markdown links in Luas docs and agent guidance."""

from __future__ import annotations

import re
import sys
from pathlib import Path
from urllib.parse import unquote, urlsplit


ROOT = Path(__file__).resolve().parents[4]
EXCLUDED_PARTS = {
    ".git",
    ".next",
    ".turbo",
    ".template",
    "coverage",
    "dist",
    "node_modules",
}
EXCLUDED_SUFFIXES = {
    ".png",
    ".jpg",
    ".jpeg",
    ".gif",
    ".webp",
    ".svg",
}
REMOTE_SCHEMES = {
    "http",
    "https",
    "mailto",
    "tel",
}

INLINE_LINK_RE = re.compile(r"!?\[[^\]\n]*\]\(([^)\n]+)\)")
REFERENCE_LINK_RE = re.compile(r"^\s{0,3}\[[^\]\n]+\]:\s+(\S+)")
INLINE_CODE_RE = re.compile(r"`[^`\n]*`")


def is_generated_or_vendor(path: Path) -> bool:
    return any(part in EXCLUDED_PARTS for part in path.relative_to(ROOT).parts)


def markdown_files() -> list[Path]:
    return sorted(
        path
        for path in ROOT.rglob("*.md")
        if not is_generated_or_vendor(path)
    )


def strip_code_fences(lines: list[str]) -> list[tuple[int, str]]:
    visible: list[tuple[int, str]] = []
    in_fence = False

    for line_number, line in enumerate(lines, start=1):
        stripped = line.lstrip()

        if stripped.startswith("```") or stripped.startswith("~~~"):
            in_fence = not in_fence
            continue

        if not in_fence:
            visible.append((line_number, line))

    return visible


def extract_raw_targets(path: Path) -> list[tuple[int, str]]:
    targets: list[tuple[int, str]] = []
    lines = path.read_text(encoding="utf-8").splitlines()

    for line_number, line in strip_code_fences(lines):
        line = INLINE_CODE_RE.sub("", line)
        targets.extend((line_number, match.group(1).strip()) for match in INLINE_LINK_RE.finditer(line))

        reference_match = REFERENCE_LINK_RE.match(line)
        if reference_match:
            targets.append((line_number, reference_match.group(1).strip()))

    return targets


def normalize_target(raw_target: str) -> str | None:
    target = raw_target.strip()

    if not target:
        return None

    if target.startswith("<") and ">" in target:
        target = target[1 : target.index(">")]
    else:
        target = target.split()[0]

    if not target or target.startswith("#"):
        return None

    parsed = urlsplit(target)
    if parsed.scheme in REMOTE_SCHEMES:
        return None

    if parsed.scheme and parsed.scheme not in {"file"}:
        return None

    if parsed.path == "":
        return None

    return unquote(parsed.path)


def resolve_target(source: Path, target: str) -> Path:
    if target.startswith("/"):
        return ROOT / target.lstrip("/")

    return source.parent / target


def main() -> int:
    failures: list[str] = []

    for source in markdown_files():
        for line_number, raw_target in extract_raw_targets(source):
            target = normalize_target(raw_target)
            if target is None:
                continue

            if Path(target).suffix.lower() in EXCLUDED_SUFFIXES:
                continue

            resolved = resolve_target(source, target)
            if not resolved.exists():
                relative_source = source.relative_to(ROOT)
                relative_target = resolved.relative_to(ROOT) if resolved.is_relative_to(ROOT) else resolved
                failures.append(f"{relative_source}:{line_number}: missing local link target: {raw_target} -> {relative_target}")

    if failures:
        print("Markdown link check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print(f"Markdown link check passed ({len(markdown_files())} files scanned).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
