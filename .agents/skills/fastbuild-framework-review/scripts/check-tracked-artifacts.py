#!/usr/bin/env python3

"""Reject artifacts committed by accident: oversized files, compiled binaries, unfinished plugin
updates, and, in the scaffold itself, installed registry plugins, which belong to projects."""

from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[4]
MAX_BYTES = 2 * 1024 * 1024
# ELF, Mach-O (32/64-bit, both byte orders, universal), and Windows PE headers.
BINARY_MAGIC = (b"\x7fELF", b"\xfe\xed\xfa\xce", b"\xfe\xed\xfa\xcf", b"\xce\xfa\xed\xfe",
                b"\xcf\xfa\xed\xfe", b"\xca\xfe\xba\xbe", b"MZ")


def main() -> int:
    tracked = subprocess.run(
        ["git", "ls-files", "-z"], cwd=ROOT, check=True, capture_output=True
    ).stdout.decode().split("\0")
    failures: list[str] = []
    for relative in filter(None, tracked):
        path = ROOT / relative
        if not path.is_file():
            continue
        size = path.stat().st_size
        if size > MAX_BYTES:
            failures.append(f"{relative} is {size // 1024} KiB; tracked files must stay under {MAX_BYTES // 1024} KiB")
            continue
        with path.open("rb") as handle:
            head = handle.read(4)
        if head.startswith(BINARY_MAGIC):
            failures.append(f"{relative} is a compiled executable; build output must not be committed")
    tracked_set = {t for t in tracked if t}
    for relative in sorted(tracked_set):
        if relative.endswith(".fastbuild-new"):
            failures.append(f"{relative} is an unmerged plugin or upgrade file; merge it and delete it")
        elif (ROOT / relative).is_file() and b"\n<<<<<<< local\n" in b"\n" + (ROOT / relative).read_bytes()[:MAX_BYTES]:
            failures.append(f"{relative} has unresolved plugin:update or project:upgrade conflict markers")
    # The scaffold ships plugins only in registry/; project:init removes it, so this runs only here.
    index_path = ROOT / "registry" / "index.json"
    if index_path.is_file():
        if "fastbuild.plugins.lock" in tracked_set:
            failures.append("fastbuild.plugins.lock is tracked: the scaffold must not keep plugins installed")
        index = json.loads(index_path.read_text(encoding="utf-8"))
        for plugin in index.get("plugins", []):
            for file in plugin.get("files", {}):
                if file in tracked_set and not file.startswith("contracts/fragments/"):
                    failures.append(f"{file} belongs to plugin {plugin['name']}; run plugin:remove {plugin['name']}")
    if failures:
        print("Tracked artifact check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1
    print(f"Tracked artifact check passed ({len([t for t in tracked if t])} files, none over {MAX_BYTES // 1024} KiB, executable, or left by a plugin).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
