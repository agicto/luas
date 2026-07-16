#!/usr/bin/env python3

"""Keep Web bundle budgets executable and performance claims semantically honest."""

from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[4]
EXPECTED_GLOBAL_BUDGET = 1_070_000
EXPECTED_ROUTE_BUDGETS = {
    "/": 600_000,
    "/login": 900_000,
    "/register": 900_000,
    "/console": 925_000,
    "/console/organizations/[organizationId]": 1_070_000,
}


def read(relative_path: str) -> str:
    return (ROOT / relative_path).read_text(encoding="utf-8")


def load_json(failures: list[str], relative_path: str) -> Any:
    path = ROOT / relative_path
    if not path.exists():
        failures.append(f"{relative_path} is missing")
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as error:
        failures.append(f"{relative_path} is invalid JSON: {error}")
        return None


def require_all(
    failures: list[str], relative_path: str, markers: tuple[str, ...]
) -> None:
    path = ROOT / relative_path
    if not path.exists():
        failures.append(f"{relative_path} is missing")
        return
    content = read(relative_path)
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
    content = read(relative_path)
    for marker in markers:
        if marker in content:
            failures.append(f"{relative_path} must not contain {marker!r}")


def check_package_scripts(failures: list[str]) -> None:
    document = load_json(failures, "web/package.json")
    if not isinstance(document, dict):
        return
    scripts = document.get("scripts")
    if not isinstance(scripts, dict):
        failures.append("web/package.json scripts must be an object")
        return

    budget_command = "node ./scripts/check-route-bundle-budget.mjs"
    for name in ("build", "build:quiet"):
        command = scripts.get(name)
        if not isinstance(command, str) or "next build" not in command:
            failures.append(f"web/package.json {name} must run next build")
        elif not command.endswith(f"&& {budget_command}"):
            failures.append(f"web/package.json {name} must finish with the route budget gate")

    if scripts.get("bundle:check") != budget_command:
        failures.append("web/package.json bundle:check must run the canonical budget checker")
    analyzer = scripts.get("bundle:analyze")
    if not isinstance(analyzer, str) or "next experimental-analyze --output" not in analyzer:
        failures.append("web/package.json bundle:analyze must use the official Turbopack analyzer")


def check_budget_policy(failures: list[str]) -> None:
    document = load_json(failures, "web/performance-budgets.json")
    if not isinstance(document, dict):
        return
    if set(document) != {"firstLoadUncompressedJavaScript"}:
        failures.append(
            "web/performance-budgets.json must own only firstLoadUncompressedJavaScript"
        )
        return
    policy = document["firstLoadUncompressedJavaScript"]
    if not isinstance(policy, dict):
        failures.append("firstLoadUncompressedJavaScript must be an object")
        return
    if policy.get("maximumAnyRouteBytes") != EXPECTED_GLOBAL_BUDGET:
        failures.append(
            f"maximumAnyRouteBytes must remain {EXPECTED_GLOBAL_BUDGET}; review governance with any intentional increase"
        )
    if policy.get("routeBudgetsBytes") != EXPECTED_ROUTE_BUDGETS:
        failures.append(
            "routeBudgetsBytes changed; update the reviewed policy and governance evidence together"
        )


def main() -> int:
    failures: list[str] = []

    check_package_scripts(failures)
    check_budget_policy(failures)

    require_all(
        failures,
        "web/scripts/check-route-bundle-budget.mjs",
        (
            ".next/diagnostics/route-bundle-stats.json",
            "firstLoadUncompressedJsBytes",
            "firstLoadChunkPaths",
            "maximumAnyRouteBytes",
            "resolveProjectFile",
            "gzipSync(source, { level: 9 })",
            "route bundle path escapes the Web project",
            "Next.js route bundle stats contain duplicate route",
            "is missing from Next.js route bundle stats",
        ),
    )
    require_all(
        failures,
        "web/src/components/theme-toggle.tsx",
        (
            "const supportedThemes = ['light', 'dark', 'system'] as const",
            "useSyncExternalStore",
            "getServerHydrationSnapshot",
            "<select",
            'data-performance-interaction="theme-selector"',
            "value={selectedTheme}",
            "setTheme(event.currentTarget.value)",
        ),
    )
    require_absent(
        failures,
        "web/src/components/theme-toggle.tsx",
        ("dropdown-menu", "<Button", "DropdownMenu"),
    )
    require_all(
        failures,
        "web/src/components/features/site/site-header-nav.tsx",
        (
            "buttonVariants({ variant: 'ghost', size: 'sm' })",
            "'interactive rounded-md max-sm:hidden'",
        ),
    )
    require_absent(
        failures,
        "web/src/components/features/site/site-header-nav.tsx",
        ("<Button",),
    )
    require_all(
        failures,
        "web/src/components/analytics.tsx",
        (
            "if (!GA_MEASUREMENT_ID) return null;",
            'strategy="lazyOnload"',
        ),
    )
    require_absent(failures, "web/src/components/analytics.tsx", ("'use client'", '"use client"'))
    require_all(
        failures,
        "web/Dockerfile",
        (
            "COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./",
            "RUN corepack enable pnpm && pnpm i --frozen-lockfile",
            "RUN mkdir -p public",
            "RUN corepack enable pnpm && pnpm run build",
            "ENV NODE_ENV=production",
            "ENV NEXT_TELEMETRY_DISABLED=1",
            "ENV HOSTNAME=0.0.0.0",
        ),
    )
    require_absent(
        failures,
        "web/Dockerfile",
        ("ENV NODE_ENV production", "ENV NEXT_TELEMETRY_DISABLED 1", 'ENV HOSTNAME "0.0.0.0"'),
    )

    require_all(
        failures,
        "web/src/test/theme-toggle.test.tsx",
        (
            "native three-state selector",
            "hydrates a persisted theme",
            "onRecoverableError",
            "getByRole('combobox'",
            "queryByRole('button')",
        ),
    )
    require_all(
        failures,
        "web/src/test/root-runtime-boundary.test.ts",
        ("strategy=\"lazyOnload\"", "if (!GA_MEASUREMENT_ID) return null;"),
    )
    require_all(
        failures,
        "web/src/test/route-accessibility-contract.test.ts",
        ("semantic and bounded on narrow screens", "max-sm:hidden"),
    )

    require_all(
        failures,
        "web/docs/PERFORMANCE.md",
        (
            "# Web Performance Boundary",
            "firstLoadUncompressedJsBytes",
            "Route performance budget",
            "Synthetic performance evidence",
            "Field Web Vitals",
            "75th percentile",
            "pnpm bundle:analyze",
            "Do not raise a budget solely to make the gate pass",
            "does not yet ship a production RUM adapter",
        ),
    )
    require_all(
        failures,
        "CONTEXT.md",
        ("**Performance budget**", "**Field Web Vital**", "Synthetic Lighthouse"),
    )
    require_all(
        failures,
        "web/README.md",
        ("docs/PERFORMANCE.md", "pnpm bundle:check", "pnpm bundle:analyze"),
    )
    require_all(
        failures,
        "web/AGENTS.md",
        ("docs/PERFORMANCE.md", "Executable route budgets", "field p75 result"),
    )
    require_all(
        failures,
        "AGENTS.md",
        ("web/docs/PERFORMANCE.md", "check-web-performance-boundary.py"),
    )
    require_all(
        failures,
        "web/.agents/skills/web-perf/SKILL.md",
        ("deterministic route", "pnpm bundle:check", "field p75"),
    )
    require_all(
        failures,
        ".agents/skills/luas-framework-review/SKILL.md",
        ("scripts/check-web-performance-boundary.py",),
    )
    require_all(
        failures,
        ".agents/skills/README.md",
        ("check-web-performance-boundary.py",),
    )
    require_all(
        failures,
        "docs/FRAMEWORK_QUALITY_ROADMAP.md",
        (
            "Completed P1 — Web Route Bundle Budget And Public Shell",
            "696,759 raw / 207,561 gzip",
            "production RUM adapter",
        ),
    )
    require_all(failures, "Makefile", ("scripts/check-web-performance-boundary.py",))

    if failures:
        print("Web performance boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print(
        "Web performance boundary check passed "
        "(route budgets, lean controls, honest evidence classes)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
