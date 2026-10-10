# Performance Baseline

Luas measures performance at two levels. Go benchmarks isolate one component; the k6 baseline
drives a release build of the whole API against PostgreSQL and fails when a hot path leaves its
latency budget. Neither is a capacity plan: a downstream product sizes its own deployment from its
own traffic.

## Component Benchmarks

| Target | Measures |
|---|---|
| `make benchmark-http` | Core HTTP middleware chain, with metrics off and on |
| `make benchmark-rate-limit` | Process-local rate limiter `Take` |
| `make benchmark-cache` | Bounded memory cache reads and churn |
| `make benchmark-workflow` | Memory workflow queue round trip |
| `make benchmark-database` | User repository query shape and list/create cost on disposable PostgreSQL |

Compare runs with `benchstat` on the same machine; absolute numbers across machines mean little.

## k6 Baseline

`make perf` builds the API, migrates and seeds a disposable database, starts the server in release
mode with JSON logs at warning level, and runs [`perf/k6/baseline.js`](../perf/k6/baseline.js):

```bash
LUAS_PERF_POSTGRES_DSN=postgres://user:pass@127.0.0.1:5432/luas_perf?sslmode=disable make perf
```

Each scenario holds a constant arrival rate, below saturation, so the numbers track per-request
cost rather than queueing:

| Scenario | Path | Rate | p95 budget |
|---|---|---:|---:|
| `readiness` | `GET /health/ready` | 50/s | 50 ms |
| `login` | `POST /v1/login` (bcrypt cost 10) | 5/s | 300 ms |
| `profile` | `GET /v1/users/profile` | 50/s | 50 ms |
| `api_keys` | `GET /v1/api-keys` | 30/s | 75 ms |
| `audit_logs` | `GET /v1/audit-logs` | 20/s | 100 ms |

Every run also requires fewer than 1% failed requests and more than 99% passing checks. On a
GitHub-hosted runner the read paths measure about 1–4 ms at p95, well inside their budgets. Login
is bcrypt-bound and measures about 165 ms against its 300 ms budget, so password-hashing cost
changes show up there first.

Both rate limiters are disabled for the run because the baseline measures request cost, not abuse
controls. Settings:

| Variable | Default | Purpose |
|---|---|---|
| `LUAS_PERF_DURATION` | `30s` | Length of each scenario |
| `LUAS_PERF_BUDGET_SCALE` | `1` | Multiplies every p95 budget, for slower hardware; error budgets are unchanged |
| `LUAS_PERF_OUTPUT` | `perf/results/summary.json` | k6 summary export |
| `K6` | `k6` on `PATH` | k6 binary |

`.github/workflows/perf.yml` runs the baseline nightly and on demand, and keeps the summary as an
artifact for 30 days.

## Profiling A Running Process

Set `SERVER_DIAGNOSTICS_ADDR=127.0.0.1:6060` to serve Go runtime profiles (`/debug/pprof/`) on a
separate listener. It is off by default, and configuration validation rejects any non-loopback
address, because profiles expose goroutine stacks, memory contents, and command-line arguments.
Reach it through a tunnel, never the public port:

```bash
kubectl port-forward deploy/luas-api 6060:6060
go tool pprof -top http://127.0.0.1:6060/debug/pprof/profile?seconds=15
```

## Connection Pool Measurements

These settings came from saturating a release build (64 concurrent clients, PostgreSQL on the same
laptop), so treat the ratios rather than the absolute numbers as the result:

| Change | `GET /v1/users/profile` | `GET /v1/api-keys` |
|---|---|---|
| Idle pool 10 (old default) | ~7,200 req/s, p99 ~100 ms | ~6,000 req/s, p99 ~60 ms |
| Idle pool = open limit (new default) | ~10,500 req/s, p99 ~13 ms | ~7,300 req/s, p99 ~20 ms |
| Plus `DB_QUERY_EXEC_MODE=cache_statement` | ~13,700 req/s, p99 ~12 ms | ~8,600 req/s, p99 ~18 ms |

With the small idle pool, the CPU profile showed new PostgreSQL connections (SCRAM key derivation)
being opened throughout the run: every dip in concurrency closed connections that the next burst
had to reopen.

## Changing The Baseline

- **New hot path.** Add a scenario and its budget in the same change as the endpoint.
- **Changing a budget.** Raise a budget only with a measured reason recorded in the commit. A
  budget that drifts upward with every regression stops measuring anything.
- **Downstream projects.** Replace the scenarios with your product's critical paths and set
  budgets from your own SLOs.
