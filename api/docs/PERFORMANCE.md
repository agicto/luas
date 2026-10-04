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

Every run also requires fewer than 1% failed requests and more than 99% passing checks. On an
Apple M-series laptop the measured p95 values are 10–20 times below these budgets. The budgets are
set for a shared CI runner, so a failure means a real regression and not noise.

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

## Changing The Baseline

- **New hot path.** Add a scenario and its budget in the same change as the endpoint.
- **Changing a budget.** Raise a budget only with a measured reason recorded in the commit. A
  budget that drifts upward with every regression stops measuring anything.
- **Downstream projects.** Replace the scenarios with your product's critical paths and set
  budgets from your own SLOs.
