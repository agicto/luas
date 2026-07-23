#!/usr/bin/env python3

"""Keep the optional cache capability bounded, atomic, and driver-neutral."""

from __future__ import annotations

import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]


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


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "api/internal/infra/cache/store.go",
        (
            "type Store interface",
            "Get(ctx context.Context, key string) ([]byte, error)",
            "SetForever(ctx context.Context",
            "Add(ctx context.Context",
            "Take(ctx context.Context, key string) ([]byte, error)",
            "DefaultMaxItemBytes = 1 << 20",
            "MaxKeyBytes = 1_024",
        ),
    )
    require_absent(
        failures,
        "api/internal/infra/cache/store.go",
        ("Has(ctx", "Increment(ctx", "Flush(ctx", "interface{}, error"),
    )

    require_all(
        failures,
        "api/internal/infra/cache/memory.go",
        (
            "DefaultMemoryMaxEntries = 10_000",
            "DefaultMemoryMaxBytes int64 = 64 << 20",
            "MaxItemBytes int",
            "payloadBytes int64",
            "makeRoomLocked",
            "moveToNewestLocked",
            "return cloneBytes(value)",
            "!now.Before(e.expiresAt)",
        ),
    )
    require_absent(
        failures,
        "api/internal/infra/cache/memory.go",
        ("go s.cleanup", "stopCleanup", "func (s *MemoryStore) Close"),
    )

    require_all(
        failures,
        "api/internal/infra/cache/redis.go",
        (
            "type RedisClient interface",
            "Namespace",
            "SetNX(ctx",
            "GetDel(ctx",
            "Unlink(ctx",
            "validRedisNamespace",
            "Redis 6.2 or later",
        ),
    )
    require_absent(
        failures,
        "api/internal/infra/cache/redis.go",
        (
            "NewRedisStoreFromConfig",
            "func (s *RedisStore) Close",
            "func (s *RedisStore) Flush",
            "json.Unmarshal",
            "json.Marshal",
        ),
    )

    require_all(
        failures,
        "api/internal/infra/cache/loader.go",
        (
            '"golang.org/x/sync/singleflight"',
            "singleflight.Group",
            "DoChan(key",
            "case <-ctx.Done()",
        ),
    )
    require_absent(
        failures,
        "api/internal/infra/cache/loader.go",
        ("var manager", "sync.Once", "func Global("),
    )

    for relative_path in (
        "api/internal/infra/cache/cache.go",
        "api/internal/infra/contracts/cache.go",
        "api/internal/infra/singleflight/singleflight.go",
    ):
        if (ROOT / relative_path).exists():
            failures.append(f"{relative_path} reintroduces a removed duplicate cache surface")

    require_all(
        failures,
        "api/internal/infra/cache/cache_contract_test.go",
        (
            "TestMemoryStoreIsBoundedAndUsesLRUEviction",
            "TestMemoryStoreAddAndTakeAreAtomic",
            "TestMemoryStoreOwnsStoredAndReturnedBytes",
            "TestLoaderWaiterCanCancelWithoutWaitingForSharedLoad",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/cache/redis_test.go",
        (
            "TestStoreContractMatchesMemoryAndRedis",
            "TestRedisStoreRealServer",
            "TestRedisStoreAddIsAtomic",
        ),
    )
    require_all(
        failures,
        "api/docs/CACHE.md",
        (
            "Cache data is a disposable copy of authoritative state",
            "`Add` is not a distributed-lock API",
            "Redis 6.2 or later",
            "does not coordinate loaders in other replicas",
        ),
    )
    require_all(
        failures,
        "api/docs/adr/0012-cache-capability-boundary.md",
        (
            "Cache Capability Boundary",
            "Switching between memory and Redis",
            "compare-and-delete release",
        ),
    )
    require_all(
        failures,
        "CONTEXT.md",
        ("**Cache capability**", "cache vs state/coordination"),
    )
    require_all(
        failures,
        "docs/SCAFFOLD_SURFACES.md",
        (
            "except packages explicitly classified as capabilities below",
            "api/internal/infra/cache",
            "check-cache-boundary.py",
        ),
    )
    require_all(
        failures,
        "Makefile",
        ("scripts/check-cache-boundary.py",),
    )
    require_all(
        failures,
        "api/AGENTS.md",
        ("docs/CACHE.md", "make benchmark-cache"),
    )
    require_all(
        failures,
        ".agents/skills/README.md",
        ("check-cache-boundary.py",),
    )
    require_all(
        failures,
        "api/Makefile",
        ("benchmark-cache:", "BenchmarkMemory(Get|GetParallel|BoundedChurn)"),
    )

    go_mod = read("api/go.mod")
    if not re.search(r"^\s*golang\.org/x/sync\s+v", go_mod, re.MULTILINE):
        failures.append("api/go.mod must declare the maintained x/sync dependency directly")

    stale_calls = ("func Global(", "func Remember(", "func Flush(", "sync.Once")
    for path in (ROOT / "api/internal/infra/cache").glob("*.go"):
        if path.name.endswith("_test.go"):
            continue
        content = path.read_text(encoding="utf-8")
        for marker in stale_calls:
            if marker in content:
                failures.append(
                    f"{path.relative_to(ROOT)} retains removed global surface {marker!r}"
                )

    if failures:
        print("Cache boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print("Cache boundary check passed (bounded bytes, atomic operations, explicit ownership).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
