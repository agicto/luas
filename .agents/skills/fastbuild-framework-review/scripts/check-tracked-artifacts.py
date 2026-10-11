#!/usr/bin/env python3

"""Reject build output committed by accident: tracked files above a size limit or compiled binaries."""

from __future__ import annotations

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
    if failures:
        print("Tracked artifact check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1
    print(f"Tracked artifact check passed ({len([t for t in tracked if t])} files, none over {MAX_BYTES // 1024} KiB or executable).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
