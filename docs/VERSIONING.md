# Versioning, Deprecation, and Releases

Luas is a scaffold that downstream projects copy and then upgrade by merging newer versions. Its
version tells a downstream team how much work an upgrade is.

## Version Numbers

Luas uses semantic versioning on the `0.y.z` line:

| Change | Version part | Requirement |
|---|---|---|
| Fix with no public behavior change | patch (`0.21.1`) | `CHANGELOG.md` entry |
| New capability, or a change a downstream project must act on | minor (`0.22.0`) | `CHANGELOG.md` and an `UPGRADING.md` entry with impact level |
| Stability commitment for public contracts | `1.0.0` | A separate decision; not before the contracts and starter catalog stop changing every minor release |

"Public behavior" means anything a downstream project can depend on: HTTP contracts and
`error_code` values, configuration variables, CLI commands, starter seams, migration names, and
generated client types.

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

Luas does not publish container images: downstream projects build and sign their own. See
[`CONTAINER_SECURITY.md`](CONTAINER_SECURITY.md) for the image evidence a downstream registry
pipeline should keep.
