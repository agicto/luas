# ADR 0013: Database Runtime And Query Baseline

- Status: Accepted
- Date: 2026-07-16

## Context

The database runtime accepted any non-`sqlite` driver name as PostgreSQL, built a keyword DSN by
interpolating credentials without escaping, and relied on GORM's automatic connection check before
performing a second unbounded `Ping`. A failed explicit ping returned without closing the pool.

The pool bounded open and idle counts and connection age, but omitted maximum idle time. Typed
configuration did not reject zero open connections, contradictory idle/open limits, malformed values,
or inert lifetime settings. PostgreSQL production could also start with a TLS-fallback mode.

Database performance claims had no repeatable PostgreSQL evidence. The default user list performed a
count and page query on every request and did not state deterministic ordering.

## Decision

1. `DatabaseConfig.Validate` owns the driver and pool invariants, while `Config.ValidateDatabase`
   adds the environment-sensitive TLS policy. Normal loading, alternate bootstraps, and database
   construction call that shared entry point. Enabled pools require a finite positive maximum,
   coherent idle limits, positive idle/lifetime/connect durations, and canonical driver/logging
   values.
2. PostgreSQL production requires a TLS-requiring `sslmode`. Local development can explicitly use
   `disable`.
3. PostgreSQL DSNs use percent-encoded connection URIs with an application name, timezone, SSL mode,
   and rounded-up server connection timeout. Secrets never appear in diagnostics.
4. GORM automatic ping is disabled. Luas configures the pool, performs one deadline-bound
   `PingContext`, and closes the pool after failed readiness.
5. The pool applies `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxIdleTime`, and
   `SetConnMaxLifetime`; process shutdown retains one explicit close owner.
6. The user list common path uses one window-count query ordered by descending primary key. Empty
   pages retain accurate totals through a fallback count.
7. An environment-gated, isolated PostgreSQL profile records application SQL statements,
   allocations, benchmark latency, and 200-sample p95. Only stable query counts are asserted; host
   timings are evidence, not an SLO.
8. GORM default write transactions and simple pgx protocol stay enabled. Prepared statement or
   transaction changes require separate pooler compatibility and correctness evidence.

## Consequences

- Misnamed drivers, malformed pool values, unlimited pools, incomplete PostgreSQL settings, and
  production non-TLS modes fail before resources are created.
- Credentials containing URI-reserved characters remain valid without custom quoting.
- Stale idle connections rotate independently from maximum age, and startup cannot wait forever for
  its explicit readiness ping.
- Normal user pages remove one database round trip and become deterministically ordered. Empty pages
  still cost two statements because total-count compatibility is retained.
- Downstream deployments must budget `DB_MAX_OPEN_CONNS` per process and explicitly configure
  production TLS.
- No schema migration or HTTP contract changes are introduced.

## References

- [Go `database/sql` package](https://pkg.go.dev/database/sql)
- [GORM configuration](https://gorm.io/docs/gorm_config.html)
- [PostgreSQL connection strings](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING)
- [pgx connection configuration](https://pkg.go.dev/github.com/jackc/pgx/v5/pgconn#ParseConfig)
