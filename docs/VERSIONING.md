# Versioning, Deprecation, and Releases

FastBuild is a scaffold that downstream projects copy and then upgrade by merging newer versions. Its
version tells a downstream team how much work an upgrade is.

## Version Numbers

FastBuild uses semantic versioning on the `0.y.z` line:

| Change | Version part | Requirement |
|---|---|---|
| Fix with no public behavior change | patch (`0.21.1`) | `CHANGELOG.md` entry |
| New capability, or a change a downstream project must act on | minor (`0.22.0`) | `CHANGELOG.md` and an `UPGRADING.md` entry with impact level |
| Stability commitment for public contracts | `1.0.0` | The compatibility promise below; see [`ROADMAP.md`](ROADMAP.md) for the exit criteria |

"Public behavior" means anything a downstream project can depend on: HTTP contracts and
`error_code` values, configuration variables, CLI commands, starter seams, migration names,
generated client types, and the plugin formats listed below.

## Compatibility Promise From 1.0

From `1.0.0`, a minor or patch release never breaks these surfaces; only a major release may, and
it ships an `UPGRADING.md` path and `project:upgrade` support for it.

| Surface | Stable within a major version | Allowed in a minor release |
|---|---|---|
| HTTP contract (`/v1`) | Paths, methods, fields, status codes, `error_code` values | New endpoints, optional fields, new error codes |
| Configuration | Variable names, meanings, and safe defaults | New variables with safe defaults |
| CLI | Command names, flags, exit codes, `--format=json` fields | New commands, flags, and JSON fields |
| Plugin manifest (`plugin.json`) and index (`version: 1`) | Every field and its meaning | New optional fields |
| Lock file (`fastbuild.plugins.lock`, `version: 1`) | Every field and its meaning | New optional fields |
| Slots | Slot names, files, and insertion semantics | New slots |
| Starter seams and migration names | Exported seams used by starters and plugins; applied migration names | New seams |
| MCP tools (`fastbuild mcp`) | Tool names and result fields | New tools and fields |

A plugin declares the oldest release it supports in `fastbuild`; the registry of each release only
lists plugins that install and pass their tests on it. `project:upgrade` moves a project between
any two releases of one major version.

Until `1.0.0`, minor releases may change these surfaces; each change has an `UPGRADING.md` entry.

## Deprecation

1. Mark the item deprecated in code and contracts, keep it working, and add an `UPGRADING.md`
   entry with the impact level `Scheduled` that names the replacement.
2. Keep it for at least one minor release after the release that announced it.
3. Remove it in a later minor release, with an `UPGRADING.md` entry that says what replaced it.

Security fixes may remove behavior immediately; the `UPGRADING.md` entry says so.

## Releasing

1. Collect accepted changes on `main` (see [`BRANCHING_AND_RELEASES.md`](BRANCHING_AND_RELEASES.md)).
2. Move the `## Unreleased` notes in `CHANGELOG.md` under `## vX.Y.Z — YYYY-MM-DD`.
3. Run `make release-check VERSION=vX.Y.Z` on an up-to-date, clean `main`. It verifies the changelog
   section, that the tag is new, and runs `make check`.
4. Tag and push: `git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z`.

Pushing the tag runs [`.github/workflows/release.yml`](../.github/workflows/release.yml). It
verifies that the tag is on `main` and has a changelog section, builds and scans both production
images, exports dependency and image SBOMs, records build provenance attestations for them, and
publishes the GitHub release with the changelog section as its notes.

FastBuild does not publish container images: downstream projects build and sign their own. See
[`CONTAINER_SECURITY.md`](CONTAINER_SECURITY.md) for the image evidence a downstream registry
pipeline should keep.
