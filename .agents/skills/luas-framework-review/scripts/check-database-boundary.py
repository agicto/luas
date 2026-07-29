#!/usr/bin/env python3

"""Keep the database runtime strict, bounded, timeout-aware, and measured."""

from __future__ import annotations

import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]

SQLITE_REFERENCE = re.compile(r"sqlite(?:3)?", re.IGNORECASE)
SOURCE_SUFFIXES = {
    ".cjs",
    ".go",
    ".js",
    ".json",
    ".mjs",
    ".sh",
    ".sql",
    ".ts",
    ".tsx",
    ".toml",
    ".yaml",
    ".yml",
}
SOURCE_FILENAMES = {"go.mod", "package.json"}
SKIPPED_PARTS = {
    ".git",
    ".next",
    "coverage",
    "dist",
    "node_modules",
}


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
            failures.append(f"{relative_path} must not contain stale marker {marker!r}")


def require_order(
    failures: list[str], relative_path: str, markers: tuple[str, ...]
) -> None:
    content = read(relative_path)
    position = -1
    for marker in markers:
        next_position = content.find(marker, position + 1)
        if next_position < 0:
            failures.append(f"{relative_path} must contain ordered marker {marker!r}")
            return
        if next_position <= position:
            failures.append(f"{relative_path} has invalid order around {marker!r}")
            return
        position = next_position


def check_sqlite_absent(failures: list[str]) -> int:
    references = 0
    for path in ROOT.rglob("*"):
        if not path.is_file() or any(part in SKIPPED_PARTS for part in path.parts):
            continue
        if path.name not in SOURCE_FILENAMES and path.suffix not in SOURCE_SUFFIXES:
            continue

        relative_path = path.relative_to(ROOT).as_posix()
        count = len(SQLITE_REFERENCE.findall(path.read_text(encoding="utf-8")))
        if count:
            failures.append(
                f"{relative_path} contains {count} SQLite reference(s); PostgreSQL is the only "
                "approved relational database target"
            )
            references += count
    return references


def main() -> int:
    failures: list[str] = []
    sqlite_references = check_sqlite_absent(failures)

    require_all(
        failures,
        "api/internal/infra/config/config.go",
        (
            "DefaultDatabaseMaxIdleConns = 10",
            "DefaultDatabaseMaxOpenConns = 100",
            "DefaultDatabaseConnMaxIdleTime = 15 * time.Minute",
            "DefaultDatabaseConnMaxLifetime = time.Hour",
            "DefaultDatabaseConnectTimeout = 5 * time.Second",
            "ConnMaxIdleTime      time.Duration",
            "ConnectTimeout       time.Duration",
            "func (d DatabaseConfig) Validate() error",
            "func (c *Config) ValidateDatabase() error",
            'd.Driver != "postgres"',
            "DB_MAX_OPEN_CONNS must be greater than 0",
            "DB_MAX_IDLE_CONNS must be between 0 and DB_MAX_OPEN_CONNS",
            "DB_CONN_MAX_IDLE_TIME must not exceed DB_CONN_MAX_LIFETIME",
            "DB_SSLMODE must require TLS in production",
            "func loadDatabaseConfig() (DatabaseConfig, error)",
            "func strictDatabaseDuration",
            "legacy integer seconds",
        ),
    )
    require_absent(
        failures,
        "api/internal/infra/config/config.go",
        (
            'env.GetInt("DB_MAX_IDLE_CONNS"',
            'env.GetInt("DB_MAX_OPEN_CONNS"',
            'time.Duration(env.GetInt("DB_CONN_MAX_LIFETIME"',
        ),
    )

    require_all(
        failures,
        "api/internal/infra/database/database.go",
        (
            "cfg.ValidateDatabase()",
            "url.UserPassword",
            "net.JoinHostPort",
            'parameters.Set("application_name"',
            'parameters.Set("connect_timeout"',
            "DisableAutomaticPing: true",
            "SetConnMaxIdleTime",
            "PingContext",
            "sqlDB.Close()",
            "postgres.New",
            "PreferSimpleProtocol: true",
        ),
    )
    require_absent(
        failures,
        "api/internal/infra/database/database.go",
        (
            'fmt.Sprintf("host=%s user=%s password=%s',
            "sqlDB.Ping()",
        ),
    )
    require_order(
        failures,
        "api/internal/infra/database/database.go",
        (
            "sqlDB.SetMaxOpenConns",
            "sqlDB.SetMaxIdleConns",
            "sqlDB.SetConnMaxIdleTime",
            "sqlDB.SetConnMaxLifetime",
            "sqlDB.PingContext",
        ),
    )

    require_all(
        failures,
        "api/internal/modules/user/repository.go",
        (
            "COUNT(*) OVER() AS page_total",
            'Order("users.id DESC")',
            "if len(rows) == 0",
            "db.Model(&UserPO{}).Count(&total)",
        ),
    )
    require_absent(
        failures,
        "api/internal/modules/user/repository.go",
        ("users.*", "users.password"),
    )
    require_all(
        failures,
        "api/internal/modules/user/postgres_profile_test.go",
        (
            'postgresProfileDSNEnv = "LUAS_TEST_POSTGRES_DSN"',
            'admin.Exec("CREATE SCHEMA " + schema)',
            'admin.Exec("DROP SCHEMA " + schema + " CASCADE")',
            "TestPostgresUserRepositoryQueryShape",
            "TestPostgresUserRepositoryPerformanceProfile",
            "BenchmarkPostgresUserRepositoryList",
            "BenchmarkPostgresUserRepositoryCreate",
            "testing.AllocsPerRun",
            "percentile95",
            "want 1",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/database/database_test.go",
        (
            "TestPostgresDSN_EncodesCredentialsAndConnectionPolicy",
            "TestNewDB_RejectsNonTLSPostgresInProduction",
            "TestNewDB_PostgresAppliesRuntimeConnectionPolicy",
            "MaxIdleTimeClosed",
            'Raw("SHOW application_name")',
        ),
    )
    require_all(
        failures,
        "api/internal/modules/user/repository_test.go",
        (
            "TestRepositoryFindAllUsesStableOrderAndPreservesEmptyPageTotal",
            "assert.NotNil(t, emptyPage)",
            "firstPageStatements.Load()",
            "emptyPageStatements.Load()",
        ),
    )

    require_all(
        failures,
        "api/.env.example",
        (
            "DB_CONN_MAX_IDLE_TIME=15m",
            "DB_CONN_MAX_LIFETIME=1h",
            "DB_CONNECT_TIMEOUT=5s",
        ),
    )
    require_absent(failures, "api/.env.example", ("DB_CONN_MAX_LIFETIME=3600",))
    require_all(
        failures,
        "api/Makefile",
        (
            "benchmark-database:",
            "LUAS_TEST_POSTGRES_DSN",
            "BenchmarkPostgresUserRepository(List|Create)",
        ),
    )
    require_all(
        failures,
        "api/docs/DATABASE.md",
        (
            "Database Runtime",
            "PostgreSQL is the only relational database compatibility target",
            "must remain absent",
            "Production rejects `disable`, `allow`, and `prefer`",
            "one application SQL statement",
            "Host, network, PostgreSQL state, and dataset size",
            "DB_MAX_OPEN_CONNS` is per process",
        ),
    )
    require_all(
        failures,
        "api/docs/adr/0013-database-runtime-and-query-baseline.md",
        (
            "Database Runtime And Query Baseline",
            "GORM automatic ping is disabled",
            "No schema migration or HTTP contract changes",
        ),
    )
    require_all(
        failures,
        "api/docs/CONFIGURATION.md",
        (
            "Database Runtime Policy",
            "PostgreSQL is the only",
            "must remain absent",
            "[`DATABASE.md`](DATABASE.md)",
        ),
    )
    require_all(
        failures,
        "api/docs/DEPLOYMENT.md",
        ("a TLS-requiring", "`DB_MAX_OPEN_CONNS` per API", "[`DATABASE.md`](DATABASE.md)"),
    )
    require_all(
        failures,
        "CONTEXT.md",
        (
            "**Database runtime**",
            "PostgreSQL is the only relational database compatibility authority",
            "Starters own their schemas",
        ),
    )
    require_all(
        failures,
        "api/AGENTS.md",
        (
            "docs/DATABASE.md",
            "make benchmark-database",
            "PostgreSQL is the only relational database compatibility target",
            "must remain absent",
        ),
    )
    require_all(
        failures,
        "api/.agents/skills/database-design/SKILL.md",
        (
            "Measure The Repository Seam",
            "make benchmark-database",
            "PostgreSQL is the only SQL compatibility target",
            "Never add SQLite",
        ),
    )
    require_all(
        failures,
        "api/.agents/skills/testing-strategy/SKILL.md",
        (
            "PostgreSQL is the only SQL compatibility target",
            "Never add SQLite",
            "LUAS_TEST_POSTGRES_DSN",
        ),
    )
    require_all(
        failures,
        ".agents/skills/systematic-debugging/SKILL.md",
        (
            "disposable PostgreSQL",
            "never substitute SQLite",
        ),
    )
    require_all(
        failures,
        "AGENTS.md",
        (
            "PostgreSQL is the only relational database compatibility target",
            "migrations, and tests must",
            "remain absent",
        ),
    )
    require_all(
        failures,
        "CONTRIBUTING.md",
        (
            "PostgreSQL as the only relational database compatibility target",
            "SQLite code or tests",
        ),
    )
    require_all(
        failures,
        "docs/FRAMEWORK_QUALITY_ROADMAP.md",
        (
            "Completed P0 — Database Runtime And Query Budget",
            "Completed P0 — PostgreSQL-Only Database Convergence",
            "zero-reference source guard",
            "one application SQL statement",
            "not an SLO or CI timing budget",
        ),
    )
    require_all(
        failures,
        ".agents/skills/README.md",
        ("check-database-boundary.py",),
    )
    require_all(
        failures,
        "Makefile",
        ("scripts/check-database-boundary.py",),
    )

    if failures:
        print("Database boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print(
        "Database boundary check passed "
        f"(PostgreSQL-only runtime; {sqlite_references} forbidden compatibility references; "
        "strict config, bounded pool, measured queries)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
