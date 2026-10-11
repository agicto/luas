#!/usr/bin/env python3

"""Keep built-in rate limits bounded, process-local, and semantically honest."""

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

    limiter_path = "api/internal/infra/ratelimit/limiter.go"
    require_all(
        failures,
        limiter_path,
        (
            "const DefaultMaxBuckets = 10_000",
            "type Limiter interface",
            "Take(ctx context.Context, key string)",
            "func WithMaxBuckets",
            "evictLeastRecentlyUsed",
            "moveToMostRecent",
            "removeExpired",
            "!now.Before(e.resetAt)",
        ),
    )
    require_absent(
        failures,
        limiter_path,
        ("go s.cleanup()", "Allow(ctx context.Context", "Hit(ctx context.Context"),
    )

    for relative_path in (
        "api/internal/infra/ratelimit/redis_store.go",
        "api/internal/infra/ratelimit/tokenscript.lua",
        "api/internal/infra/redis/redis.go",
    ):
        if (ROOT / relative_path).exists():
            failures.append(f"{relative_path} reintroduces the removed unassembled Redis limiter surface")

    require_all(
        failures,
        "api/internal/infra/config/config.go",
        (
            'env.GetInt("MIDDLEWARE_RATE_LIMIT_MAX_BUCKETS", DefaultRateLimitMaxBuckets)',
            'env.GetInt("AUTH_RATE_LIMIT_MAX_BUCKETS_PER_RULE", DefaultRateLimitMaxBuckets)',
            "MIDDLEWARE_RATE_LIMIT_MAX_BUCKETS must be greater than 0",
            "AUTH_RATE_LIMIT_MAX_BUCKETS_PER_RULE must be greater than 0",
        ),
    )
    require_absent(
        failures,
        "api/internal/infra/config/config.go",
        ("type RedisConfig struct", 'env.Get("REDIS_HOST"'),
    )
    require_all(
        failures,
        "api/internal/bootstrap/http.go",
        ("MaxBuckets: cfg.Middleware.RateLimit.MaxBuckets",),
    )
    require_all(
        failures,
        "api/internal/modules/user/auth_abuse_guard.go",
        (
            "cfg.MaxBucketsPerRule",
            "ratelimit.WithMaxBuckets(maxBuckets)",
            '"auth:" + string(endpoint) + ":ip:"',
            '"auth:" + string(endpoint) + ":subject:"',
        ),
    )
    require_all(
        failures,
        "api/.env.example",
        (
            "MIDDLEWARE_RATE_LIMIT_MAX_BUCKETS=10000",
            "AUTH_RATE_LIMIT_MAX_BUCKETS_PER_RULE=10000",
        ),
    )
    env_example = read("api/.env.example")
    if re.search(r"^REDIS_(?:HOST|PORT|PASSWORD|DB)=", env_example, re.MULTILINE):
        failures.append("api/.env.example must not advertise unassembled REDIS_* runtime settings")

    require_all(
        failures,
        "api/docs/MIDDLEWARE.md",
        (
            "bounded, process-local fixed-window limiter",
            "least recently used bucket",
            "does not ship a built-in Redis rate-limit driver",
            "silently fall back to independent per-process buckets",
        ),
    )
    require_all(
        failures,
        "api/docs/adr/0011-rate-limit-runtime-boundary.md",
        (
            "Rate-Limit Runtime Boundary",
            "Silent fallback to",
            "LRU eviction and fixed-window boundary bursts",
        ),
    )
    require_all(
        failures,
        "contracts/AUTHENTICATION.md",
        (
            "AUTH_RATE_LIMIT_MAX_BUCKETS_PER_RULE",
            "explicitly assembled shared limiter",
            "built-in Redis rate-limit driver",
        ),
    )

    if "golang.org/x/time" in read("api/go.mod"):
        failures.append("api/go.mod must not retain x/time solely for the removed Redis limiter")

    if failures:
        print("Rate-limit boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print("Rate-limit boundary check passed (bounded local baseline; shared enforcement explicit).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
