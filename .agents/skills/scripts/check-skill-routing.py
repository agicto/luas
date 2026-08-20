#!/usr/bin/env python3
"""Validate recorded forward-routing cases against repository skill policy."""

from __future__ import annotations

import argparse
import csv
import hashlib
import sys
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class Skill:
    name: str
    explicit_only: bool
    description: str


def repository_root() -> Path:
    return Path(__file__).resolve().parents[3]


def read_frontmatter(skill_file: Path) -> tuple[str, str]:
    name = ""
    description = ""
    for line in skill_file.read_text(encoding="utf-8").splitlines():
        if line.startswith("name: "):
            name = line.removeprefix("name: ").strip()
        elif line.startswith("description: "):
            description = line.removeprefix("description: ").strip()
        elif line == "---" and name:
            break
    if not name or not description:
        raise ValueError(f"{skill_file}: missing name or description")
    return name, description


def load_skills(root: Path) -> dict[str, Skill]:
    skills: dict[str, Skill] = {}
    for scope in (root / ".agents/skills", root / "api/.agents/skills", root / "web/.agents/skills"):
        for skill_file in sorted(scope.glob("*/SKILL.md")):
            if ".template" in skill_file.parts:
                continue
            name, description = read_frontmatter(skill_file)
            metadata = skill_file.parent / "agents/openai.yaml"
            explicit_only = "allow_implicit_invocation: false" in metadata.read_text(encoding="utf-8")
            if name in skills:
                raise ValueError(f"duplicate skill name: {name}")
            skills[name] = Skill(name, explicit_only, description)
    return skills


def metadata_fingerprint(skills: dict[str, Skill]) -> str:
    payload = "".join(
        f"{skill.name}\t{str(skill.explicit_only).lower()}\t{skill.description}\n"
        for skill in sorted(skills.values(), key=lambda item: item.name)
    )
    return hashlib.sha256(payload.encode("utf-8")).hexdigest()


def parse_args() -> argparse.Namespace:
    root = repository_root()
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "cases",
        nargs="?",
        type=Path,
        default=root / ".agents/skills/evals/routing-cases.tsv",
    )
    parser.add_argument(
        "--fingerprint",
        type=Path,
        default=root / ".agents/skills/evals/routing-metadata.sha256",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    root = repository_root()
    skills = load_skills(root)
    errors: list[str] = []
    fingerprint_file = args.fingerprint
    expected_fingerprint = fingerprint_file.read_text(encoding="utf-8").strip()
    actual_fingerprint = metadata_fingerprint(skills)
    if expected_fingerprint != actual_fingerprint:
        errors.append(
            "routing metadata changed; review the affected cases and replace "
            f"{fingerprint_file} with {actual_fingerprint}"
        )
    expected_coverage: set[str] = set()
    boundary_coverage: set[str] = set()
    seen_ids: set[str] = set()
    total = correct = false_positive = false_negative = misroute = 0

    with args.cases.open(encoding="utf-8", newline="") as handle:
        reader = csv.DictReader(handle, delimiter="\t")
        required = ["id", "mode", "expected_skill", "observed_skill", "boundary_for", "prompt"]
        if reader.fieldnames != required:
            print(f"error: header must be exactly: {', '.join(required)}", file=sys.stderr)
            return 1
        for line_number, row in enumerate(reader, start=2):
            total += 1
            case_id = row["id"].strip()
            mode = row["mode"].strip()
            expected = row["expected_skill"].strip()
            observed = row["observed_skill"].strip()
            boundary = row["boundary_for"].strip()
            prompt = row["prompt"].strip()

            if not case_id or case_id in seen_ids:
                errors.append(f"line {line_number}: missing or duplicate id {case_id!r}")
            seen_ids.add(case_id)
            if mode not in {"implicit", "explicit"}:
                errors.append(f"{case_id}: mode must be implicit or explicit")
            for field, value in (("expected_skill", expected), ("observed_skill", observed)):
                if value != "none" and value not in skills:
                    errors.append(f"{case_id}: unknown {field} {value!r}")
            if not prompt:
                errors.append(f"{case_id}: prompt is empty")

            if expected in skills:
                expected_coverage.add(expected)
                if mode == "implicit" and skills[expected].explicit_only:
                    errors.append(f"{case_id}: explicit-only expected skill used in implicit mode")
                if mode == "explicit" and f"${expected}" not in prompt:
                    errors.append(f"{case_id}: explicit prompt must name ${expected}")
            if observed in skills and mode == "implicit" and skills[observed].explicit_only:
                errors.append(f"{case_id}: implicit observation selected explicit-only skill {observed}")

            if boundary != "-":
                if boundary not in skills or not skills[boundary].explicit_only:
                    errors.append(f"{case_id}: boundary_for must name an explicit-only skill")
                else:
                    boundary_coverage.add(boundary)
                if mode != "implicit" or expected == boundary or observed == boundary:
                    errors.append(f"{case_id}: boundary case leaked or expected {boundary}")

            if expected == observed:
                correct += 1
            elif expected == "none":
                false_positive += 1
            elif observed == "none":
                false_negative += 1
            else:
                misroute += 1

    missing_positive = sorted(set(skills) - expected_coverage)
    missing_boundaries = sorted(
        {name for name, skill in skills.items() if skill.explicit_only} - boundary_coverage
    )
    if missing_positive:
        errors.append(f"missing positive cases: {', '.join(missing_positive)}")
    if missing_boundaries:
        errors.append(f"missing explicit-only boundary cases: {', '.join(missing_boundaries)}")
    explicit_count = sum(skill.explicit_only for skill in skills.values())
    if total < len(skills) + explicit_count + 5:
        errors.append("routing suite needs full skill coverage plus boundary and routine cases")

    accuracy = (correct / total * 100) if total else 0
    print(
        "Skill routing forward test: "
        f"{total} cases, {correct} correct, {false_positive} false positives, "
        f"{false_negative} false negatives, {misroute} misroutes, {accuracy:.1f}% recorded accuracy"
    )
    print(
        f"Coverage: {len(expected_coverage)}/{len(skills)} skills, "
        f"{len(boundary_coverage)}/{explicit_count} explicit-only boundaries"
    )
    for error in errors:
        print(f"error: {error}", file=sys.stderr)
    return 1 if errors or false_positive or false_negative or misroute else 0


if __name__ == "__main__":
    raise SystemExit(main())
