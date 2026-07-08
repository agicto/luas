# Branching and Release Governance

This document defines how Luas branches map to testing, deployment, and release decisions.
It is process guidance for the scaffold itself and for teams using Luas as a starter kit.

## Goals

- Keep `main` releasable.
- Let many feature branches test together without forcing all of them into a release.
- Make release content selectable by PR or commit, not by whatever happens to be on `dev`.
- Keep deployment branches mechanical so they do not become hidden integration branches.

## Branch Roles

| Branch pattern | Role | Merge direction |
|---|---|---|
| `main` | Production truth. Every commit must be reviewed, verified, and releasable. | Receives only accepted `release/*`, `hotfix/*`, or fully verified feature PRs. |
| `feature/<name>` | Short-lived work branch for one starter, feature, capability, doc, or scaffold improvement. | Opens PRs into `dev` for shared testing and into `release/*` or `main` only after acceptance. |
| `dev` | Shared testing branch and integration sandbox. It may contain unfinished work. | Never merge `dev` back to `main`. Reset or recreate it from `main` when needed. |
| `dev-c` | Secondary testing branch for an alternate testing environment. It follows the same rules as `dev`. | Never merge `dev-c` back to `main`. |
| `deploy-dev` / `deploy-dev-c` | Mechanical deployment trigger branches managed by CI. | Only CI updates them from `dev` / `dev-c`; humans do not use them as source branches. |
| `release/<date-or-version>` | Selected release candidate assembled from `main` plus accepted changes. | Merges to `main` after verification; fixes flow back to active feature branches when needed. |
| `hotfix/<name>` | Urgent production fix from `main`. | Merges to `main`; then sync or cherry-pick to active `release/*`, `dev`, or feature branches as needed. |

## Default Flow

1. Create a `feature/<name>` branch from the latest `main`.
2. Open a PR to `dev` when the feature needs shared environment testing.
3. Keep unfinished work on the feature branch or on `dev`; do not let it define the release.
4. Create `release/<date-or-version>` from `main` when release content is known.
5. Merge accepted feature PRs into the release branch, or cherry-pick specific commits when the feature branch has extra unfinished work.
6. Run verification on the release branch.
7. Merge the release branch into `main`, tag the release, and deploy from `main` or the production deployment pipeline.

```text
main
  |-- feature/a -> dev test sandbox -> release/2026-07-08 -> main
  |-- feature/b -> dev test sandbox
  `-- feature/c -> dev-c test sandbox -> release/2026-07-08 -> main
```

## Release Candidate Rules

- Create `release/*` from `main`, not from `dev`.
- Include only accepted changes.
- Prefer merging a full feature branch when that branch contains exactly the release-ready work.
- Prefer cherry-picking when a feature branch contains both accepted and unfinished commits.
- Do not add unrelated refactors while stabilizing a release branch.
- Only fix release-blocking issues on `release/*`.
- After a release fix lands, sync the fix back to the owning feature branch if that branch remains active.

## CI and Environment Mapping

- `main`, `dev`, and `dev-c` run the full CI workflow.
- `dev` and `dev-c` trigger CI-managed deployment branches through `.github/workflows/sync-deploy-branches.yml`.
- `deploy-dev` and `deploy-dev-c` are deployment triggers, not collaboration branches.
- If a future production deployment branch is added, document its source of truth here before enabling CI writes.

## Protection Rules

- Protect `main` with required PR review and green CI.
- Block direct pushes to `main` except for emergency repository-owner actions.
- Protect `release/*` with the same verification requirements as `main`.
- Allow `dev` to move quickly, but keep its role explicit: it proves compatibility, not release readiness.
- When a branch contains product-specific downstream app work, it must stay outside the Luas scaffold repository unless the work is a generic starter-kit improvement.

## Verification Before Release

Run these before merging `release/*` or `hotfix/*` to `main`:

```bash
make check
bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh
bash .agents/skills/luas-framework-review/scripts/check-api-boundaries.sh
bash .agents/skills/scripts/validate-skill.sh --all
```

For contract-sensitive changes, also verify the affected API and mock BFF behavior against
[`../contracts/README.md`](../contracts/README.md).

## Anti-Patterns

- Merging `dev` into `main`.
- Treating `dev` as a release candidate after unrelated work has entered it.
- Using deployment trigger branches as human working branches.
- Releasing a branch because it is green while ignoring whether its content was selected.
- Hiding product-specific downstream app behavior inside Luas scaffold branches.
