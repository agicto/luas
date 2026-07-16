#!/usr/bin/env python3

"""Keep shared Web control composition semantics and guidance aligned."""

from __future__ import annotations

import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]


def require_all(
    failures: list[str], relative_path: str, markers: tuple[str, ...]
) -> None:
    path = ROOT / relative_path
    if not path.exists():
        failures.append(f"{relative_path} is missing")
        return

    content = path.read_text(encoding="utf-8")
    for marker in markers:
        if marker not in content:
            failures.append(f"{relative_path} must contain {marker!r}")


def require_absent(
    failures: list[str], relative_path: str, markers: tuple[str, ...]
) -> None:
    path = ROOT / relative_path
    if not path.exists():
        failures.append(f"{relative_path} is missing")
        return

    content = path.read_text(encoding="utf-8")
    for marker in markers:
        if marker in content:
            failures.append(f"{relative_path} must not contain {marker!r}")


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "web/src/components/ui/button.tsx",
        (
            "Slot, Slottable",
            "@radix-ui/react-slot",
            "aria-disabled:pointer-events-none",
            "const isDisabled = disabled || loading",
            "<Slottable>{composedChild}</Slottable>",
            "aria-busy={loading ? true : ariaBusy}",
            "aria-disabled={isDisabled ? true : ariaDisabled}",
            "tabIndex={isDisabled ? -1 : tabIndex}",
            "prepareComposedChild",
            "onClickCapture: preventDisabledClick",
            "preventDisabledClick",
        ),
    )
    require_absent(
        failures,
        "web/src/components/ui/button.tsx",
        (
            "disabled={props.disabled || loading}",
            "const Comp = asChild ? Slot",
        ),
    )
    require_all(
        failures,
        "web/src/test/button-composition.test.tsx",
        (
            "applies the button contract directly to the composed link",
            "keeps leading and trailing content inside the composed link hit area",
            "preserves native disabled and loading semantics for a button",
            "makes a disabled composed link inert without invalid HTML attributes",
            "keeps loading feedback inside an inert composed link",
            "expect(container.firstElementChild).toBe(link)",
            "expect(fireEvent.click(link)).toBe(false)",
            'aria-disabled="false"',
            "tabIndex={0}",
            "childClickCapture",
            "childKeyDownCapture",
        ),
    )
    require_all(
        failures,
        "web/AGENTS.md",
        (
            "#### Composed Control Contract",
            "`Button asChild`",
            "`Slottable`",
            "`src/test/button-composition.test.tsx`",
        ),
    )
    require_all(
        failures,
        "web/.agents/skills/accessibility-audit/SKILL.md",
        (
            "#### Luas Composed Control Contract",
            "`Button asChild`",
            "`aria-disabled`",
            "`aria-busy`",
        ),
    )
    require_all(
        failures,
        "web/.agents/skills/testing-standards/SKILL.md",
        ("`src/test/button-composition.test.tsx`", "semantic host"),
    )
    require_all(
        failures,
        "Makefile",
        ("check-web-ui-primitive-boundary.py",),
    )

    if failures:
        print("Web UI primitive boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print("Web UI primitive boundary check passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
