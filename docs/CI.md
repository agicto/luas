# Continuous Integration

Luas treats CI as part of the scaffold architecture. Workflows must be reproducible, least-privilege,
compatible with supported runners, and reviewable without trusting a movable action tag.

## Workflow Roles

| Workflow | Responsibility | Default token permission |
|---|---|---|
| `ci.yml` | Root governance, API build/lint/test/race gates, and Web type/lint/test/build gates | `contents: read` |
| `container.yml` | Production image contract and local Compose lifecycle | `contents: read` |
| `skill-self-test.yml` | Starter-module validators and repository Skill metadata | `contents: read` |
| `sync-deploy-branches.yml` | Mechanical `dev` / `dev-c` deployment-branch synchronization | `contents: write` |

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
The container workflow additionally requires a working Docker daemon with Compose v2.

The action runtime is separate from the Web project runtime: CI currently tests the Web application
on Node 22. The pnpm version comes only from `web/package.json` `packageManager`; the setup action
receives `package_json_file: web/package.json` instead of duplicating that version in workflow YAML.

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
6. Run `make governance` and `make check`, then verify the pushed workflow itself on GitHub.
7. Inspect annotations as well as pass/fail status; a successful run with a deprecation warning is not
   considered a clean upgrade.

## Verification

```bash
python3 .agents/skills/luas-framework-review/scripts/check-ci-actions.py
make governance
make check
```

Local checks prove workflow structure and application behavior. Only the remote runs prove that the
pinned action commits execute on the configured runner and that GitHub emits no runtime-deprecation
annotation.
