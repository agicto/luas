#!/usr/bin/env python3

"""Validate Luas registered-route JSON without third-party packages."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any


KIND = "luas.route_catalog"
SCHEMA_VERSION = 1
TOP_LEVEL_FIELDS = {"kind", "schema_version", "active_starters", "routes"}
ROUTE_FIELDS = {"method", "path"}
METHODS = {
    "GET",
    "POST",
    "PUT",
    "PATCH",
    "DELETE",
    "HEAD",
    "OPTIONS",
    "CONNECT",
    "TRACE",
}
STARTER_PATTERN = re.compile(r"^[a-z][a-z0-9_-]*$")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("catalog", type=Path)
    parser.add_argument("--expect-starter", action="append", default=[])
    parser.add_argument(
        "--require-route",
        action="append",
        default=[],
        nargs=2,
        metavar=("METHOD", "PATH"),
    )
    return parser.parse_args()


def is_string_list(value: Any) -> bool:
    return isinstance(value, list) and all(isinstance(item, str) for item in value)


def validate(
    document: Any, expected_starters: list[str], required_routes: list[list[str]]
) -> list[str]:
    failures: list[str] = []
    if not isinstance(document, dict):
        return ["catalog must be a JSON object"]
    if set(document) != TOP_LEVEL_FIELDS:
        failures.append(
            f"top-level fields are {sorted(document)}, expected {sorted(TOP_LEVEL_FIELDS)}"
        )
    if document.get("kind") != KIND:
        failures.append(f"kind must be {KIND!r}")
    if document.get("schema_version") != SCHEMA_VERSION:
        failures.append(f"schema_version must be {SCHEMA_VERSION}")

    starters = document.get("active_starters")
    if not is_string_list(starters):
        failures.append("active_starters must be an array of strings")
        starters = []
    else:
        if len(starters) != len(set(starters)):
            failures.append("active_starters must be unique")
        for starter in starters:
            if not STARTER_PATTERN.fullmatch(starter):
                failures.append(f"active starter {starter!r} is not canonical")
    if expected_starters and starters != expected_starters:
        failures.append(
            f"active_starters are {starters!r}, expected {expected_starters!r}"
        )

    raw_routes = document.get("routes")
    if not isinstance(raw_routes, list) or not raw_routes:
        failures.append("routes must be a non-empty array")
        raw_routes = []

    routes: list[tuple[str, str]] = []
    seen: set[tuple[str, str]] = set()
    for index, route in enumerate(raw_routes):
        if not isinstance(route, dict):
            failures.append(f"routes[{index}] must be an object")
            continue
        if set(route) != ROUTE_FIELDS:
            failures.append(
                f"routes[{index}] fields are {sorted(route)}, expected {sorted(ROUTE_FIELDS)}"
            )
        method = route.get("method")
        path = route.get("path")
        if method not in METHODS:
            failures.append(f"routes[{index}].method {method!r} is unsupported")
        if not isinstance(path, str) or not path.startswith("/"):
            failures.append(f"routes[{index}].path {path!r} must be absolute")
        if method in METHODS and isinstance(path, str) and path.startswith("/"):
            identity = (method, path)
            if identity in seen:
                failures.append(f"route {method} {path} is duplicated")
            seen.add(identity)
            routes.append(identity)

    expected_order = sorted(routes, key=lambda route: (route[1], route[0]))
    if routes != expected_order:
        failures.append("routes must be sorted by path and then method")

    for raw_method, path in required_routes:
        method = raw_method.upper()
        if (method, path) not in seen:
            failures.append(f"required route {method} {path} is missing")

    return failures


def main() -> int:
    args = parse_args()
    try:
        document = json.loads(args.catalog.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        print(f"Route catalog validation failed: {error}", file=sys.stderr)
        return 1

    failures = validate(document, args.expect_starter, args.require_route)
    if failures:
        print("Route catalog validation failed:", file=sys.stderr)
        for failure in failures:
            print(f"- {failure}", file=sys.stderr)
        return 1

    print(
        "Validated route catalog "
        f"v{SCHEMA_VERSION} with {len(document['active_starters'])} active starters "
        f"and {len(document['routes'])} routes."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
