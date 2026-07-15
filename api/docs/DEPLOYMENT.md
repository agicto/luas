# Container and Deployment Contract

Luas provides a production-oriented API image and a local Compose stack. It does not choose a
cloud, registry, rollout controller, secret store, or migration orchestrator for downstream apps.

## Surface Ownership

| Surface | Owner | Contract |
|---|---|---|
| `Dockerfile` | API core | Builds the production image and defines runtime-safe defaults. |
| `docker-compose.yml` | Local development | Runs the API and PostgreSQL on loopback with replaceable local credentials. |
| `scripts/verify-container.sh` | Verification | Builds and exercises the production image contract. |
| `scripts/verify-compose.sh` | Verification | Rebuilds the local worktree or reuses one explicitly verified image, then exercises PostgreSQL and startup. |
| `.github/workflows/container.yml` | CI | Runs the same container verifier when API or container sources change. |
| Production deployment manifests | Downstream app | Own secrets, network policy, replicas, migrations, rollout, and rollback. |

## Production Image

The image uses a multi-stage Go build and the distroless non-root runtime. It deliberately does not
copy `.env.example` or any `.env` file. Missing production configuration must fail at startup rather
than silently falling back to development values.

Image defaults:

```text
APP_ENV=production
APP_DEBUG=false
SERVER_MODE=release
SERVER_HOST=0.0.0.0
LOG_LEVEL=info
LOG_STDOUT=true
LOG_FILE_ENABLED=false
LOG_JSON=true
```

The image health check executes `/app/luas health:check`, which probes
`http://127.0.0.1:${SERVER_PORT}/health/live` with a two-second deadline. Override the target only
with `HEALTHCHECK_URL` when the process genuinely listens elsewhere.

Build and verify the complete runtime contract:

```bash
make container-check
make compose-check
```

The verifier checks the non-root user, image health configuration, liveness, database-disabled
readiness, JSON request logs on stdout, absence of `/app/.env`, and a zero exit code after SIGTERM.
Standalone `make compose-check` always rebuilds the current worktree so a stale local tag cannot
produce a false green result. In CI, the Compose verifier receives the explicit image tag built and
checked by the immediately preceding container step; a missing explicit image fails instead of
silently building a different artifact. It then starts PostgreSQL on random loopback ports, runs the
explicit local migration opt-in, waits for readiness, and completes a real starter registration.
When `OPTIONAL_STARTERS` contains `organization`, the same check also exercises PostgreSQL-backed
organization creation plus invitation create, duplicate conflict, list, revoke, and replacement
semantics without requiring an external email provider. It verifies the active organization CORS
preflight, required-header error, owner resolution, non-member non-disclosure, and current member
role against PostgreSQL. It then creates registered-user membership
fixtures and exercises the PII-minimized member directory, owner-only role mutation, account-delete
guard, admin removal, previous-owner leave, and a concurrent two-request ownership transfer. The
transfer gate requires exactly one `200`, one `403`, one persisted owner, and the previous owner
demoted to `admin`. A second concurrency gate races account deletion against organization creation;
either operation may win, but their status pair must be `201/409` or `204/404`, and a direct
PostgreSQL check must find zero memberships attached to soft-deleted users.

## Local Compose

`docker-compose.yml` is a development convenience, not a production manifest. It exposes API and
PostgreSQL ports on `127.0.0.1` by default, uses visibly local-only credentials, disables the optional
AI provider, and explicitly starts the single local API process with `luas serve --migrate` after
PostgreSQL becomes healthy. This local opt-in is not the production migration strategy.

```bash
docker compose up --build --wait
docker compose down
```

Override local ports with `LUAS_API_PORT` and `LUAS_DB_PORT`. Override local credentials with
`JWT_SECRET` and `LUAS_DB_PASSWORD`. Set `OPTIONAL_STARTERS=organization` to exercise the optional
ownership kernel; the API process and its local startup migration receive the same value.
`ORGANIZATION_INVITATION_TTL` is forwarded to the API container and defaults to `168h`.
`docker compose down --volumes` also deletes local database data.

## Production Inputs

A downstream production deployment must inject at least:

- `JWT_SECRET`: generated secret with at least 32 characters.
- `CORS_ALLOW_ORIGINS`: explicit production browser origins.
- `CORS_ALLOW_HEADERS`: retain `Authorization` and `Organization-Id` when a cross-origin browser
  calls active-organization routes; Luas includes both in its default allow-list.
- `DB_DRIVER`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USERNAME`, and `DB_PASSWORD` when the default
  database-backed starters remain installed.
- `SERVER_TRUSTED_PROXIES`: only exact ingress/load-balancer IPs or CIDRs when forwarding headers are
  trusted.
- `OPTIONAL_STARTERS`: one identical additive selection for every API replica, migration job, and
  seeder job. Omit or set empty when no optional starter is enabled.

Keep secrets in the deployment platform's secret store, not in the image, Compose file, repository,
or command history. Keep `/health/live` as the process liveness signal and `/health/ready` as the
traffic-readiness signal. The Docker image health check intentionally uses liveness so a temporary
database outage does not create a restart loop; an orchestrator should remove unready replicas from
traffic based on readiness.

Production logs belong on structured stdout. Collection, retention, rotation, compression, and
external storage are deployment responsibilities; the local file handler is not a production log
shipping system. See [`CONFIGURATION.md`](CONFIGURATION.md) for the typed configuration lifecycle.

Run schema migration as an explicit pre-deploy job using the same image and production environment:

```bash
docker run --rm --entrypoint /app/luas <runtime-env-arguments> <image> migrate --force
```

Do not run migrations independently in every application replica. The downstream deployment owns
serialization, failure handling, and rollback policy. Database mutation commands derive production
mode from the validated configuration snapshot; `db:migrate`, `db:rollback`, `db:reset`, and
`db:fresh` require an explicit `--force` in production. `serve --migrate` is rejected in production
because startup replicas are not a migration serialization mechanism. A mismatch in
`OPTIONAL_STARTERS` between the pre-deploy job and serving replicas is a deployment contract
violation: it can produce routes without tables or tables without owning runtime behavior.

## Change Checklist

When changing container behavior:

1. Keep image defaults production-safe and Compose defaults local-only.
2. Do not copy environment files or credentials into the image.
3. Keep `health:check`, the Docker `HEALTHCHECK`, and actual health routes aligned.
4. Keep request logs visible through container stdout when file logging is disabled.
5. Run `make container-check`, `docker compose config --quiet`, and the normal API verification tier.
6. Run `make compose-check` when Compose, migrations, database startup, or default starters change.
