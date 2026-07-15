---
name: deployment
description: Deploy and verify the Luas API image, local Compose stack, health probes, logs, migrations, and runtime configuration.
---

# Luas API Deployment

## Purpose

Use the repository's actual container contract when changing the Dockerfile, Compose stack, runtime
environment, probes, migrations, container logging, or deployment CI. Luas supplies a scaffold image;
downstream apps own their cloud, registry, secret store, rollout, and rollback.

## Read First

1. `../../../../CONTEXT.md` for scaffold and downstream-app vocabulary.
2. `../../../docs/DEPLOYMENT.md` for the canonical container/deployment contract.
3. `../../../Dockerfile`, `../../../docker-compose.yml`, and
   `../../../scripts/verify-container.sh` for executable behavior.
4. `../../../docs/MIDDLEWARE.md` when proxy trust, metrics exposure, or HTTP transport changes.

## Surface Rules

### Production Image

- Keep the final stage distroless and non-root.
- Never copy `.env`, `.env.example`, credentials, private keys, or local configuration into the image.
- Keep production/release mode, wildcard container bind, JSON stdout logs, and disabled file logging
  explicit in the image.
- Use `/health/live` for the image liveness check. Do not use readiness as a restart signal.
- Keep the health command shell-free so it runs in the distroless image.

### Local Compose

- Treat `docker-compose.yml` as a local development stack only.
- Bind published API and database ports to loopback by default.
- Mark bundled credentials as local-only and allow environment overrides.
- Do not describe Compose as a production manifest or silently add production rollout policy.

### Production Deployment

- Inject production CORS origins, database/provider credentials, and trusted proxy ranges through
  the deployment platform. Authentication sessions use typed lifetime policy and no signing secret.
- Keep liveness and readiness separate: liveness detects a stuck process; readiness controls traffic.
- Run migrations as one explicit pre-deploy job. Do not let every replica race to migrate on startup.
- Keep TLS, distributed rate limits, network policy, replica count, autoscaling, and secret rotation
  deployment-owned.

## Workflow

1. Classify the change as image, local Compose, probe, log, migration, CI, or downstream deployment.
2. Update the owning runtime seam before documentation.
3. Add focused Go or shell regression coverage for new behavior.
4. Run the container verifier locally when Docker is available.
5. Update `docs/DEPLOYMENT.md`, README, environment examples, and this skill when semantics change.
6. Run normal API verification and inspect the remote Container workflow after pushing.

## Commands

```bash
cd api
make container-check
make compose-check
docker compose config --quiet
docker compose up --build --wait
docker compose down
go test ./internal/infra/console/commands ./pkg/logger ./internal/infra/config
```

`make container-check` must prove all of the following:

- the image runs as non-root;
- Docker has an executable health check;
- liveness returns 200 and DB-disabled readiness returns 503;
- request logs reach container stdout as JSON;
- `/app/.env` is absent;
- SIGTERM produces a zero exit code.

## Review Checklist

- Does the image fail safely when required production configuration is missing?
- Can a local Compose run start without pretending to be production?
- Can the runtime probe execute without a shell or curl in the image?
- Are request logs visible through `docker logs` when the filesystem is read-only or unwritable?
- Are build artifacts, environment files, logs, test binaries, and coverage files excluded from context?
- Is a migration change paired with the SQL migration review skill and a deployment serialization plan?
- Are measured image/context claims reported as local evidence rather than universal budgets?

## Pair With

- `sql-migration-review` for schema rollout safety.
- `logging-standards` for log schema or handler changes.
- `verification-before-completion` before reporting the slice complete.
- Root `luas-framework-review` when deployment ownership or scaffold semantics change.
