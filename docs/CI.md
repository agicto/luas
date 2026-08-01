# Continuous Integration

Luas treats CI as part of the scaffold architecture. Workflows must be reproducible, least-privilege,
compatible with supported runners, and reviewable without trusting a movable action tag.

## Workflow Roles

| Workflow | Responsibility | Default token permission |
|---|---|---|
| `ci.yml` | Root governance, OpenAPI lint/generation/route/breaking gates, API build/lint/test/runtime-route/race gates, PostgreSQL 15-18 compatibility, plus Next.js Web and Admin Console type/lint/test/build gates | `contents: read` |
| `container.yml` | API image identity, smoke test, SBOM/scan evidence, and local Compose lifecycle | `contents: read` |
| `dependency-security.yml` | Scheduled and change-triggered OSV lockfile scan plus CycloneDX SBOM artifact | `contents: read` |
| `skill-self-test.yml` | Starter-module validators and repository Skill metadata | `contents: read` |
| `sync-deploy-branches.yml` | Mechanical `dev` / `dev-c` deployment-branch synchronization | `contents: write` |
| `web-container.yml` | Web image identity, smoke test, SBOM, and vulnerability gate | `contents: read` |

Only the deployment-branch sync workflow has write access. Do not add write permissions to validation
workflows. Do not use `pull_request_target` for repository code verification; it combines privileged
base-repository context with untrusted pull-request input. New workflow files require an explicit
permission-policy entry in `check-ci-actions.py` and a role in the table above.

## Runner Contract

Workflows default to `ubuntu-latest`. An installation may set repository variable
`CI_RUNNER_LABELS` to a JSON array of labels for a compatible self-hosted runner.

The reviewed JavaScript actions use the Node 24 action runtime. Self-hosted runners must run GitHub
Actions Runner `v2.327.1` or newer. `actions/checkout` intentionally remains on the latest reviewed
v5 patch because v6 raises the runner requirement to `v2.329.0` for its credential-storage change.
The API container workflow additionally requires a working Docker daemon with Compose v2. The Web
container workflow requires the same Docker/buildx contract but does not run Compose.

The container job verifies one artifact identity. `verify-container.sh` builds and checks
`luas-api:ci`; the following Compose step passes that tag explicitly and must fail if it is absent.
The typed-setting Compose step reuses the same immutable image with
`OPTIONAL_STARTERS=organization,setting` and exercises its PostgreSQL/API/CLI lifecycle.
The usage-metering Compose step reuses that image with `OPTIONAL_STARTERS=organization,usage` and
proves exact replay, conflicting idempotency rejection, concurrent quota serialization, durable
denials, private user/organization reads, account cleanup, pruning, and migration rollback/reapply.
Standalone `make compose-check` has no explicit tag and therefore rebuilds the current worktree,
preventing a stale local image from producing a false green result.

Both image workflows emit maximal BuildKit metadata, validate reviewed material digests, export a
CycloneDX 1.7 image SBOM, and enforce the Trivy HIGH/CRITICAL, secret, and EOL gate. Their build
metadata and SBOM artifacts are retained for 14 days. These are unsigned CI evidence; registry
attestations and Cosign identity remain a downstream publication responsibility. See
[`CONTAINER_SECURITY.md`](CONTAINER_SECURITY.md).

The Web image smoke check also requests `/login` from the running standalone server and validates
the centralized production browser-security response policy. This proves image/runtime wiring; the
unit test and root governance check separately own policy semantics and Next.js Proxy conventions.

The action runtime is separate from browser project runtimes: CI tests both `web/` and `admin/`
on Node 22 and Node 24. Node 22 remains the production image and type-definition baseline. Each
pnpm version comes from that project's `packageManager`; the setup action receives its exact
`package_json_file` instead of duplicating the version in workflow YAML. Both workspaces require
the same exact pnpm version, so an older developer-global binary cannot silently rewrite or
interpret either lockfile.

The API job runs `make route-catalog-check` after its build/lint/test tier. That command assembles
the real configured runtime, emits schema-versioned JSON, validates its closed shape and ordering,
and requires core plus default-starter routes. Route inventory therefore cannot pass CI from a
parallel source parser that omits health, conditional metrics, or optional starter registration.

The complete API gate runs on PostgreSQL 16. A separate compatibility matrix uses immutable images
for PostgreSQL 15, 17, and 18 and runs `make test-postgres-compatibility`. That focused command owns
migrations, repositories, transaction and locking behavior, optional starter persistence, and
PostgreSQL durable tasks without repeating unrelated browser or pure-Go checks. The support window
is documented in [`../api/docs/DATABASE.md`](../api/docs/DATABASE.md) and must move forward before an
upstream major version reaches end of life.

The HTTP Contracts job validates OpenAPI 3.1, checks committed TypeScript output in both browser
shells, and proves every described operation exists in the real Go route assembly. Pull requests
compare the target-commit schema with the proposed schema using pinned `oasdiff` 1.11.7; breaking
changes fail instead of silently narrowing downstream clients. Initial OpenAPI adoption is allowed
when the target commit has no schema.

## Dependency Supply Chain

The Dependency Security workflow calls the same root script available to developers. It downloads
OSV-Scanner 2.3.8 from the official release, verifies the platform asset by SHA-256, scans
`api/go.mod` plus the contracts, Web, and Admin Console pnpm lockfiles, validates a CycloneDX 1.5
inventory, and uploads that inventory for 14 days. It uses read-only repository permission and does
not depend on private-repository GitHub Advanced Security availability.

`make governance` checks the tool/version/digest pins, pnpm resolution and build-script policy,
lockfile integrity coverage, Dependabot ecosystems, CI trigger, and time-bounded OSV exceptions.
`make dependency-scan` is the live network-backed vulnerability gate. See
[`DEPENDENCY_SECURITY.md`](DEPENDENCY_SECURITY.md) for update, privacy, license, and exception rules.

## Action Supply Chain

Every external `uses:` reference must use a full-length commit SHA with the reviewed release tag in
an adjacent comment:

```yaml
- uses: actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd # v5.0.1
```

A full-length commit SHA is the immutable execution identity. The version comment is human-readable
provenance, not the value executed by the runner. Local actions under `./` are exempt from remote pin
rules. Docker actions are not implicitly trusted: a `docker://` reference fails governance until an
immutable digest policy is deliberately introduced. New external action repositories require an
explicit review and an allowlist update in `check-ci-actions.py`.

## Updating An Action

1. Read the official release notes and breaking changes.
2. Confirm the release action metadata uses a supported runtime and record its minimum runner.
3. Resolve the release tag to the commit in the action's own repository; peel annotated tags.
4. Update every occurrence to the same full SHA and exact release comment.
5. Update the allowlist in `check-ci-actions.py` and this runner contract when requirements change.
6. Run `make check`, then verify the pushed workflow itself on GitHub.
7. Inspect annotations as well as pass/fail status; a successful run with a deprecation warning is not
   considered a clean upgrade.

## Verification

```bash
python3 .agents/skills/luas-framework-review/scripts/check-ci-actions.py
python3 .agents/skills/luas-framework-review/scripts/check-dependency-supply-chain.py
python3 .agents/skills/luas-framework-review/scripts/check-container-supply-chain.py
python3 .agents/skills/luas-framework-review/scripts/check-route-contract-discovery.py
cd api && make route-catalog-check
make dependency-scan
make check
```

Local checks prove workflow structure and application behavior. Only the remote runs prove that the
pinned action commits execute on the configured runner and that GitHub emits no runtime-deprecation
annotation.
