#!/usr/bin/env python3

"""Check scaffold-level HTTP error_code contracts across docs, API, and Web."""

from __future__ import annotations

import re
import sys
from dataclasses import dataclass
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]
CONTRACTS_README = ROOT / "contracts" / "README.md"
API_RESPONSE_CODES = ROOT / "api" / "pkg" / "response" / "error_codes.go"
API_DOMAIN_CODES = ROOT / "api" / "internal" / "domain" / "error_codes.go"
WEB_CODES = ROOT / "web" / "src" / "http" / "codes.ts"


@dataclass(frozen=True)
class ContractError:
    status: int
    error_code: str


CODE_VALUE = r"[A-Z][A-Z0-9_]*(?:\.[A-Z][A-Z0-9_]*)+"
CONTRACT_ROW_RE = re.compile(rf"^\|\s*(\d{{3}})\s*\|\s*`({CODE_VALUE})`\s*\|")
GO_CONST_RE = re.compile(rf"^\s*([A-Za-z][A-Za-z0-9_]+)\s*=\s*\"({CODE_VALUE})\"")
TS_API_CODE_RE = re.compile(
    rf"^\s*([A-Z][A-Z0-9_]+):\s*'({CODE_VALUE})',",
    re.MULTILINE,
)
TS_STATUS_MAP_RE = re.compile(r"^\s*(\d{3}):\s*ApiErrorCode\.([A-Z][A-Z0-9_]+),")


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


def parse_web_api_codes() -> dict[str, str]:
    codes: dict[str, str] = {}
    source = read(WEB_CODES)
    try:
        block = source.split("export const ApiErrorCode = {", 1)[1].split("} as const;", 1)[0]
    except IndexError:
        return codes

    for match in TS_API_CODE_RE.finditer(block):
        codes[match.group(1)] = match.group(2)

    return codes


def parse_web_status_map() -> dict[int, str]:
    api_codes = parse_web_api_codes()
    status_map: dict[int, str] = {}
    in_status_map = False

    for line in read(WEB_CODES).splitlines():
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
    web_api_codes = set(parse_web_api_codes().values())
    web_status_map = parse_web_status_map()
    documented_by_status: dict[int, set[str]] = {}

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

    missing_web = sorted(contract_codes - web_api_codes)
    if missing_web:
        failures.append(f"web/src/http/codes.ts ApiErrorCode is missing contract error_code values: {', '.join(missing_web)}")

    missing_domain_web = sorted(api_domain_codes - web_api_codes)
    if missing_domain_web:
        failures.append(
            "web/src/http/codes.ts ApiErrorCode is missing API domain error_code values: "
            + ", ".join(missing_domain_web)
        )

    for status, documented_codes in sorted(documented_by_status.items()):
        mapped_code = web_status_map.get(status)
        if mapped_code is None:
            failures.append(f"web/src/http/codes.ts HttpStatusErrorCodeMap is missing HTTP {status}")
            continue

        if mapped_code not in documented_codes:
            failures.append(
                f"web/src/http/codes.ts maps HTTP {status} to {mapped_code}, "
                f"but contracts/README.md documents {', '.join(sorted(documented_codes))}"
            )

    undocumented_statuses = sorted(set(web_status_map) - set(documented_by_status))
    if undocumented_statuses:
        failures.append(
            "web/src/http/codes.ts maps undocumented HTTP statuses: "
            + ", ".join(str(status) for status in undocumented_statuses)
        )

    if failures:
        print("Error contract check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print(
        "Error contract check passed "
        f"({len(contract_codes)} contract error_code values, {len(api_domain_codes)} API domain values, "
        f"{len(web_status_map)} Web status fallbacks)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
