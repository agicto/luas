# Database Runtime

Luas treats the database connection as core runtime infrastructure. `internal/infra/database` owns
GORM construction, the process-wide `database/sql` pool, startup readiness, diagnostics, and
shutdown. Each starter still owns its tables, migrations, transactions, query shape, ordering, and
pagination semantics.

## Dialect Authority

PostgreSQL is the only relational database compatibility target for Luas. SQLite runtime code,
drivers, dependencies, DSNs, dialect branches, fixtures, migrations, and tests must remain absent.
GORM portability and an in-memory database pass do not prove PostgreSQL query, constraint,
transaction, locking, index, or migration behavior.

Keep pure service tests database-free through existing seams and test doubles. Tests that exercise
SQL behavior use an isolated schema in a caller-supplied disposable PostgreSQL database, following
the `LUAS_TEST_POSTGRES_DSN` profile below.

## Supported Versions

Luas 1.0 targets PostgreSQL 15, 16, 17, and 18 on their latest minor releases. PostgreSQL 16 runs
the complete API CI gate; a focused compatibility matrix runs migrations, repositories,
transactions, locks, and durable tasks on 15, 17, and 18. PostgreSQL 14 is intentionally outside the
1.0 support window because its community support ends in November 2026. Support follows upstream
security maintenance rather than an unbounded "or later" promise. Review the
[PostgreSQL versioning policy](https://www.postgresql.org/support/versioning/) when updating this
matrix.

```bash
LUAS_TEST_POSTGRES_DSN='postgres://user:password@127.0.0.1:5432/luas_test?sslmode=disable' \
  make test-postgres-compatibility
```

## Typed Configuration

The database settings are parsed once by `internal/infra/config` and validated before GORM creates
resources:

| Variable | Default | Runtime contract |
|---|---:|---|
| `DB_DRIVER` | `postgres` | Exact `postgres` is required; every other value fails validation. |
| `DB_MAX_OPEN_CONNS` | `100` | Positive and finite; zero cannot silently enable an unlimited pool. |
| `DB_MAX_IDLE_CONNS` | `10` | Zero through `DB_MAX_OPEN_CONNS`. |
| `DB_CONN_MAX_IDLE_TIME` | `15m` | Positive and no longer than connection lifetime. |
| `DB_CONN_MAX_LIFETIME` | `1h` | Positive maximum reuse age. |
| `DB_CONNECT_TIMEOUT` | `5s` | Positive startup connection and ping deadline. |
| `DB_SLOW_THRESHOLD` | `1s` | Positive slow-query logging threshold. |

Duration values should use Go units such as `500ms`, `15m`, or `1h`. Legacy integer values remain
accepted as seconds for `DB_CONN_MAX_LIFETIME` and the other database duration fields, but new
deployments should use units. Malformed integers, booleans, and durations fail loading even when the
database is disabled, so a typo cannot silently select a default.

PostgreSQL additionally requires a host, port, database, username, password, timezone, and one of
the supported `sslmode` values. Production rejects `disable`, `allow`, and `prefer`; use `require`,
`verify-ca`, or preferably `verify-full` with deployment-owned trust material. Local Compose remains
explicitly non-TLS because its traffic stays inside the local development stack.

## Connection And Pool Ownership

PostgreSQL settings are encoded as a connection URI rather than interpolated into a keyword string.
This preserves spaces and reserved characters in credentials and database names, handles IPv6, and
keeps the DSN out of errors and logs. The URI includes `application_name`, `connect_timeout`,
`sslmode`, and `timezone`.

GORM automatic ping is disabled. Luas applies the finite pool policy first, then performs exactly one
`PingContext` with `DB_CONNECT_TIMEOUT`. A failed ping closes the newly created pool before startup
returns an error. The HTTP kernel closes the long-lived shared pool after request draining and trace
shutdown.

The PostgreSQL adapter deliberately retains `PreferSimpleProtocol`. Do not enable GORM
`PrepareStmt`, pgx implicit statement caching, or `SkipDefaultTransaction` globally without a
transaction audit and measurements against the deployment's pooler mode. GORM write transactions
protect consistency; a local latency win is not enough to remove them across all starters.

## Starter Query Baseline

The default user repository is the representative database profile:

- `FindAll` selects only response-owned columns, excluding password hashes and soft-delete metadata,
  then uses `COUNT(*) OVER()` and `ORDER BY users.id DESC` to return a normal page and its total in
  one application SQL statement. An empty or out-of-range page runs one fallback count so the
  existing total contract remains accurate.
- `Create` remains one application SQL statement and keeps GORM's default transaction behavior.
- Statement counts describe SQL emitted through GORM and exclude transaction-control protocol such
  as `BEGIN` and `COMMIT`.

The profile creates a generated schema in a caller-supplied disposable PostgreSQL database, migrates
and seeds only that schema, then closes its pool and drops the schema with `CASCADE`. The database
role therefore needs temporary schema create/drop permission. It never truncates or migrates the
caller's existing schemas.

```bash
LUAS_TEST_POSTGRES_DSN='postgres://user:password@127.0.0.1:5432/luas_profile?sslmode=disable' \
  make benchmark-database
```

The command checks the production `NewDB` path, exact query shape, 200-sample p95 profile, Go
allocations, and five two-second benchmark runs. Host, network, PostgreSQL state, and dataset size
affect latency. Query-count assertions are regression guards; timings and allocations are comparison
evidence until repeated CI data justifies a portable budget.

## Operations

Watch `database/sql.DBStats`, especially connection wait count/duration and idle/lifetime closures,
alongside PostgreSQL saturation. `DB_MAX_OPEN_CONNS` is per process: multiply it by the maximum API,
worker, and command replicas before comparing it with server or pooler capacity. Raising the number
without capacity evidence can move queuing from the process into PostgreSQL and make tail latency
worse.

The readiness checker pings with its caller deadline. A disabled or unavailable database remains
down for readiness rather than being reported as healthy. Database errors and DSNs must never be
returned in public HTTP responses.

Architecture rationale is recorded in
[`adr/0013-database-runtime-and-query-baseline.md`](adr/0013-database-runtime-and-query-baseline.md).
