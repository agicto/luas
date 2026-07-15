#!/usr/bin/env python3

"""Keep API runtime configuration behind the typed startup authority."""

from __future__ import annotations

import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]
ENV_IMPORT = "github.com/zgiai/luas/api/pkg/env"


def read(relative_path: str) -> str:
    return (ROOT / relative_path).read_text(encoding="utf-8")


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


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "api/docs/CONFIGURATION.md",
        (
            "one typed configuration authority",
            "Configuration changes require a process restart",
            "does not serialize a configuration cache",
            "LUAS_ENV_FILE",
        ),
    )
    require_all(
        failures,
        "api/cmd/server/main.go",
        (
            "config.Load()",
            "bootstrap.InitLogger(cfg)",
            "wiring.InitApplicationWithConfig(cfg)",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/config/config.go",
        (
            "type Config struct",
            "func (c *Config) IsProduction() bool",
            'case "production", "prod", "release":',
            "func Load() (*Config, error)",
            "func LoadAIConfig()",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/console/commands/migrate.go",
        ("requireProductionForce", 'force := slices.Contains(args, "--force")'),
    )
    require_all(
        failures,
        "api/internal/infra/console/commands/serve.go",
        ("startup migrations are disabled in production", "cfg.IsProduction()"),
    )

    api_root = ROOT / "api"
    allowed_env_imports = {api_root / "internal/infra/config/config.go"}
    for path in api_root.rglob("*.go"):
        if path.name.endswith("_test.go"):
            continue
        if ENV_IMPORT in path.read_text(encoding="utf-8") and path not in allowed_env_imports:
            failures.append(
                f"{path.relative_to(ROOT)} imports pkg/env outside the typed config authority"
            )

    allowed_process_reads = {
        api_root / "pkg/env/env.go",
        api_root / "internal/infra/plugin/loader.go",  # operating-system PATH discovery
    }
    process_read = re.compile(r"os\.(?:Getenv|LookupEnv)\(")
    runtime_source_roots = (
        api_root / "cmd/server",
        api_root / "cmd/luas",
        api_root / "internal",
        api_root / "pkg",
    )
    for source_root in runtime_source_roots:
        for path in source_root.rglob("*.go"):
            if path.name.endswith("_test.go") or path in allowed_process_reads:
                continue
            if process_read.search(path.read_text(encoding="utf-8")):
                failures.append(
                    f"{path.relative_to(ROOT)} reads process configuration outside pkg/env"
                )

    forbidden_paths = (
        "api/internal/infra/config/dynamic.go",
        "api/internal/infra/config/watcher.go",
        "api/pkg/logger/clickhouse.go",
    )
    for relative_path in forbidden_paths:
        if (ROOT / relative_path).exists():
            failures.append(f"{relative_path} reintroduces a removed configuration bypass")

    forbidden_symbols = re.compile(
        r"\b(?:LoadDynamic|ConfigDynamic|WatchEnvFile|CacheConfig|ClearCache)\s*\("
    )
    for path in api_root.rglob("*.go"):
        if path.name.endswith("_test.go"):
            continue
        match = forbidden_symbols.search(path.read_text(encoding="utf-8"))
        if match:
            failures.append(
                f"{path.relative_to(ROOT)} reintroduces removed symbol {match.group(0)!r}"
            )

    go_mod = read("api/go.mod")
    if "github.com/fsnotify/fsnotify" in go_mod:
        failures.append("api/go.mod must not depend on fsnotify without a real reload lifecycle")

    env_example = read("api/.env.example")
    removed_keys = (
        "JWT_SECRET",
        "JWT_EXPIRE_DAYS",
        "LOG_CH_ENABLED",
        "LOG_MAX_SIZE",
        "LOG_MAX_AGE",
        "LOG_MAX_BACKUPS",
        "LOG_COMPRESS",
    )
    for key in removed_keys:
        if re.search(rf"^{key}=", env_example, re.MULTILINE):
            failures.append(f"api/.env.example advertises unsupported key {key}")

    if failures:
        print("Configuration authority check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print("Configuration authority check passed (typed startup snapshot is canonical).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
