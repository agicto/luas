#!/usr/bin/env python3

"""Check scaffold-level HTTP error_code contracts across docs, API, and browser shells."""

from __future__ import annotations

import re
import sys
from dataclasses import dataclass
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]
CONTRACTS_README = ROOT / "contracts" / "README.md"
API_RESPONSE_CODES = ROOT / "api" / "pkg" / "response" / "error_codes.go"
API_DOMAIN_CODES = ROOT / "api" / "internal" / "domain" / "error_codes.go"
OPENAPI_CONTRACT = ROOT / "contracts" / "openapi.yaml"
BROWSER_CODE_PATHS = {
    "web": ROOT / "web" / "src" / "http" / "codes.ts",
    "web-spa": ROOT / "web-spa" / "src" / "http" / "codes.ts",
}


@dataclass(frozen=True)
class ContractError:
    status: int
    error_code: str


CODE_VALUE = r"[A-Z][A-Z0-9_]*(?:\.[A-Z][A-Z0-9_]*)+"
CONTRACT_ROW_RE = re.compile(rf"^\|\s*(\d{{3}})\s*\|\s*`({CODE_VALUE})`\s*\|")
GO_CONST_RE = re.compile(rf"^\s*([A-Za-z][A-Za-z0-9_]+)\s*=\s*\"({CODE_VALUE})\"")
TS_API_CODE_RE = re.compile(
    rf"\b([A-Z][A-Z0-9_]+):\s*'({CODE_VALUE})',",
    re.MULTILINE,
)
TS_STATUS_MAP_RE = re.compile(r"^\s*(\d{3}):\s*ApiErrorCode\.([A-Z][A-Z0-9_]+),")
OPENAPI_ENUM_VALUE_RE = re.compile(rf"^\s{{8}}-\s+({CODE_VALUE})\s*$")


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def parse_contract_errors() -> list[ContractError]:
    errors: list[ContractError] = []

    for line in read(CONTRACTS_README).splitlines():
        match = CONTRACT_ROW_RE.match(line)
        if match:
            errors.append(ContractError(status=int(match.group(1)), error_code=match.group(2)))

    return errors


def parse_go_codes(path: Path) -> dict[str, str]:
    codes: dict[str, str] = {}

    for line in read(path).splitlines():
        match = GO_CONST_RE.match(line)
        if match:
            codes[match.group(1)] = match.group(2)

    return codes


def parse_browser_api_codes(path: Path) -> dict[str, str]:
    codes: dict[str, str] = {}
    source = read(path)
    try:
        block = source.split("export const ApiErrorCode = {", 1)[1].split(
            "export type ApiErrorCodeValue", 1
        )[0]
    except IndexError:
        return codes

    for match in TS_API_CODE_RE.finditer(block):
        codes[match.group(1)] = match.group(2)

    return codes


def parse_browser_status_map(path: Path) -> dict[int, str]:
    api_codes = parse_browser_api_codes(path)
    status_map: dict[int, str] = {}
    in_status_map = False

    for line in read(path).splitlines():
        if line.startswith("export const HttpStatusErrorCodeMap"):
            in_status_map = True
            continue

        if in_status_map and line.startswith("};"):
            break

        if in_status_map:
            match = TS_STATUS_MAP_RE.match(line)
            if match:
                status = int(match.group(1))
                key = match.group(2)
                status_map[status] = api_codes.get(key, f"<missing ApiErrorCode.{key}>")

    return status_map


def parse_openapi_error_codes() -> set[str]:
    codes: set[str] = set()
    in_error_code = False
    in_enum = False

    for line in read(OPENAPI_CONTRACT).splitlines():
        if line == "    ErrorCode:":
            in_error_code = True
            continue
        if in_error_code and line == "      enum:":
            in_enum = True
            continue
        if in_enum:
            match = OPENAPI_ENUM_VALUE_RE.match(line)
            if match:
                codes.add(match.group(1))
                continue
            break

    return codes


def duplicate_values(values: list[str]) -> list[str]:
    seen: set[str] = set()
    duplicates: set[str] = set()

    for value in values:
        if value in seen:
            duplicates.add(value)
        seen.add(value)

    return sorted(duplicates)


def main() -> int:
    failures: list[str] = []
    contract_errors = parse_contract_errors()
    contract_codes = {error.error_code for error in contract_errors}
    api_response_codes = set(parse_go_codes(API_RESPONSE_CODES).values())
    api_domain_codes = set(parse_go_codes(API_DOMAIN_CODES).values())
    openapi_codes = parse_openapi_error_codes()
    browser_api_codes = {
        name: set(parse_browser_api_codes(path).values())
        for name, path in BROWSER_CODE_PATHS.items()
    }
    browser_status_maps = {
        name: parse_browser_status_map(path)
        for name, path in BROWSER_CODE_PATHS.items()
    }
    documented_by_status: dict[int, set[str]] = {}

    expected_openapi_codes = api_response_codes | api_domain_codes
    missing_openapi = sorted(expected_openapi_codes - openapi_codes)
    extra_openapi = sorted(openapi_codes - expected_openapi_codes)
    if missing_openapi:
        failures.append("contracts/openapi.yaml ErrorCode is missing API values: " + ", ".join(missing_openapi))
    if extra_openapi:
        failures.append("contracts/openapi.yaml ErrorCode has values absent from the API: " + ", ".join(extra_openapi))

    for error in contract_errors:
        documented_by_status.setdefault(error.status, set()).add(error.error_code)

    if not contract_errors:
        failures.append("contracts/README.md must document scaffold-level error rows")

    duplicate_contract_codes = duplicate_values([error.error_code for error in contract_errors])
    if duplicate_contract_codes:
        failures.append(f"contracts/README.md has duplicate error_code rows: {', '.join(duplicate_contract_codes)}")

    missing_api = sorted(contract_codes - api_response_codes)
    if missing_api:
        failures.append(f"api/pkg/response/error_codes.go is missing contract error_code values: {', '.join(missing_api)}")

    for name, path in BROWSER_CODE_PATHS.items():
        relative_path = path.relative_to(ROOT)
        api_codes = browser_api_codes[name]
        status_map = browser_status_maps[name]

        missing_contract = sorted(contract_codes - api_codes)
        if missing_contract:
            failures.append(
                f"{relative_path} ApiErrorCode is missing contract error_code values: "
                + ", ".join(missing_contract)
            )

        missing_domain = sorted(api_domain_codes - api_codes)
        if missing_domain:
            failures.append(
                f"{relative_path} ApiErrorCode is missing API domain error_code values: "
                + ", ".join(missing_domain)
            )

        if api_codes != openapi_codes:
            missing_schema = sorted(openapi_codes - api_codes)
            extra_schema = sorted(api_codes - openapi_codes)
            if missing_schema:
                failures.append(
                    f"{relative_path} ApiErrorCode is missing OpenAPI values: " + ", ".join(missing_schema)
                )
            if extra_schema:
                failures.append(
                    f"{relative_path} ApiErrorCode has values absent from OpenAPI: " + ", ".join(extra_schema)
                )

        for status, documented_codes in sorted(documented_by_status.items()):
            mapped_code = status_map.get(status)
            if mapped_code is None:
                failures.append(f"{relative_path} HttpStatusErrorCodeMap is missing HTTP {status}")
                continue

            if mapped_code not in documented_codes:
                failures.append(
                    f"{relative_path} maps HTTP {status} to {mapped_code}, "
                    f"but contracts/README.md documents {', '.join(sorted(documented_codes))}"
                )

        undocumented_statuses = sorted(set(status_map) - set(documented_by_status))
        if undocumented_statuses:
            failures.append(
                f"{relative_path} maps undocumented HTTP statuses: "
                + ", ".join(str(status) for status in undocumented_statuses)
            )

    next_codes = browser_api_codes["web"]
    spa_codes = browser_api_codes["web-spa"]
    if next_codes != spa_codes:
        only_next = sorted(next_codes - spa_codes)
        only_spa = sorted(spa_codes - next_codes)
        if only_next:
            failures.append(
                "web-spa/src/http/codes.ts is missing Next.js Web ApiErrorCode values: "
                + ", ".join(only_next)
            )
        if only_spa:
            failures.append(
                "web/src/http/codes.ts is missing static SPA ApiErrorCode values: "
                + ", ".join(only_spa)
            )

    if failures:
        print("Error contract check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print(
        "Error contract check passed "
        f"({len(contract_codes)} contract error_code values, {len(openapi_codes)} OpenAPI values, "
        f"{len(api_domain_codes)} API domain values, "
        + ", ".join(
            f"{len(browser_status_maps[name])} {name} status fallbacks"
            for name in BROWSER_CODE_PATHS
        )
        + ")."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
