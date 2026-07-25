#!/usr/bin/env python3

"""Keep repository documentation and source comments in English."""

from __future__ import annotations

import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]
HAN = re.compile(r"[\u3400-\u4dbf\u4e00-\u9fff]")
COMMENT_WITH_HAN = re.compile(r"(?://|#|/\*|\*)[^\n]*[\u3400-\u4dbf\u4e00-\u9fff]")
SOURCE_SUFFIXES = {".css", ".go", ".js", ".mjs", ".py", ".sh", ".ts", ".tsx", ".yaml", ".yml"}
IGNORED_PARTS = {".git", ".next", "coverage", "dist", "node_modules"}


def is_localization_source(path: Path) -> bool:
    relative = path.relative_to(ROOT).as_posix()
    return (
        relative == "web-spa/src/i18n/resources.ts"
        or relative == "web/src/i18n/locales.ts"
        or (relative.startswith("web/src/i18n/modules/") and relative.endswith("/zh-Hans.ts"))
    )


def is_test_source(path: Path) -> bool:
    relative = path.relative_to(ROOT).as_posix()
    return path.name.endswith("_test.go") or "/src/test/" in f"/{relative}"


def main() -> int:
    failures: list[str] = []

    for path in sorted(ROOT.rglob("*")):
        if not path.is_file() or any(part in IGNORED_PARTS for part in path.parts):
            continue
        if path.suffix != ".md" and path.suffix not in SOURCE_SUFFIXES and path.name != ".editorconfig":
            continue

        content = path.read_text(encoding="utf-8")
        relative = path.relative_to(ROOT).as_posix()

        if path.suffix == ".md" or path.name == ".editorconfig":
            matcher = HAN
        elif is_localization_source(path):
            continue
        elif is_test_source(path):
            matcher = COMMENT_WITH_HAN
        else:
            matcher = HAN

        for number, line in enumerate(content.splitlines(), start=1):
            if matcher.search(line):
                failures.append(f"{relative}:{number}: translate documentation or comments to English")

    if failures:
        for failure in failures:
            print(f"English source check failed: {failure}", file=sys.stderr)
        return 1

    print("English source check passed (locales and Unicode test values preserved).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
