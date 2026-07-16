# Container Supply-Chain Security

Luas treats each production image as a deployable artifact with four distinct controls: immutable
build inputs, runtime verification, image inventory and vulnerability policy, and downstream
publication identity. These controls complement the source dependency policy in
[`DEPENDENCY_SECURITY.md`](DEPENDENCY_SECURITY.md); one must not be presented as another.

## Semantic Model

| Term | Repository evidence | Guarantee |
|---|---|---|
| Base image identity | Exact tag plus multi-platform `sha256` index digest in each `Dockerfile` | A build cannot silently consume a different base under the same tag. |
| Build evidence | Buildx metadata JSON emitted with maximal BuildKit provenance | Records the output digest, Dockerfile entry point, invocation, and reviewed material digests. |
| Image SBOM | Trivy CycloneDX 1.7 JSON | Inventories the packages visible in the built runtime image, not merely source lockfiles. |
| Vulnerability gate | Trivy 0.72.0 image scan | Fails on HIGH/CRITICAL vulnerabilities, detected secrets, or a known end-of-life base OS. |
| Publication signature | Downstream registry plus Cosign/OIDC policy | Binds a pushed registry digest to a trusted publishing identity. Luas does not invent that identity. |

Build metadata and an SBOM are evidence about an artifact. They are not signatures. The CI artifacts
in this repository are deliberately **unsigned** validation outputs retained for 14 days. A
downstream application that publishes an image must attach provenance/SBOM evidence to its registry
digest and apply its own Cosign identity and verification policy.

## Immutable Build Inputs

Both Dockerfiles pin the Dockerfile frontend and every external `FROM` to an exact readable tag plus
the reviewed multi-platform digest:

- API builder: Go 1.25.12 on Alpine 3.24;
- API runtime: distroless static Debian 12 `nonroot`;
- Web build/runtime source: Node 22.23.1 on Alpine 3.24.

The Web final image materializes a cleaned runtime root into a new `scratch` stage. apk, npm, npx,
Corepack, pnpm, Yarn, Node build headers, debugger helpers, and manual pages are absent from the
resulting filesystem, while the Node and native Sharp artifacts keep the same musl ABI used during
the build. This removes development attack surface without creating an Alpine/glibc runtime mismatch.

Dependabot discovers Docker updates weekly. A base-image update is not complete until the tag,
Dockerfile digest, verifier material digest, governance constant, smoke test, and image scan change
together. A digest-only update still requires release-note and image-diff review.

## Local Verification

Build and exercise both runtime contracts:

```bash
cd api && make container-check
cd web && bash scripts/verify-container.sh luas-web:container-check
```

The verifiers use `docker buildx build --load`, write OCI source/revision/version labels, request
maximal BuildKit metadata, and check that provenance contains every reviewed material digest. They
then verify non-root execution, health behavior, absence of embedded environment files, and bounded
SIGTERM termination. The Web verifier additionally checks the exact Node version and confirms that
development tooling and every `.env*` file are absent.

Scan an already-built image and export its image-level SBOM:

```bash
IMAGE=luas-api:container-check make container-scan
IMAGE=luas-api:container-check make container-sbom

IMAGE=luas-web:container-check make container-scan
SBOM_OUTPUT="$PWD/luas-web.cdx.json" IMAGE=luas-web:container-check make container-sbom
```

`scripts/container-security.sh` downloads the official Trivy 0.72.0 archive over HTTPS, verifies a
reviewed platform-specific SHA-256, and re-extracts the executable from that verified archive on
every run. The vulnerability database remains live by design; do not describe a successful scan as
permanent evidence against advisories published later.

Image SBOM identity is portable across Docker stores: the validator requires a named container
subject, the requested image in Trivy `Reference`/`RepoTag`, and a SHA-256 Trivy `ImageID`. An OCI
`purl` is validated when a registry or containerd store provides one, but it is not fabricated for
standard daemon-local images.

`make check` runs the deterministic container governance rules, but intentionally does not build or
network-scan images. The dedicated image workflows own those Docker- and network-backed gates.

## CI Evidence

`.github/workflows/container.yml` builds, smoke-tests, inventories, and scans the API image before
running Compose scenarios. `.github/workflows/web-container.yml` applies the same image contract to
the Web deployable. Each read-only workflow uploads two files keyed by the exact Git commit:

- `<unit>.build-metadata.json`, containing the output digest and maximal BuildKit material evidence;
- `<unit>.cdx.json`, containing the CycloneDX 1.7 runtime inventory and recorded advisories.

Artifacts remain available for 14 days and upload even when the scan step fails, unless the job is
cancelled. This preserves evidence needed to diagnose a failed gate. A green source dependency scan
does not replace these image scans because OS packages and runtime-only content do not live in
`go.mod` or `pnpm-lock.yaml`.

## Scan Policy And Exceptions

The blocking policy covers HIGH/CRITICAL vulnerabilities, secrets, and known EOL operating systems.
It does not pass `--ignore-unfixed`: remediation availability changes urgency, not the existence of a
finding. Lower-severity findings remain visible in the SBOM for review without turning every advisory
database update into an unplanned release stop.

The default [`.trivyignore.yaml`](../.trivyignore.yaml) is empty. A temporary exception must name one
exact finding `id` and include both a non-empty `statement` and a future `expired_at` date. Wildcards,
severity-wide suppression, non-expiring entries, and scanner failures are forbidden. Root governance
rejects incomplete, duplicate, wildcard, or expired entries. Prefer upgrading or removing the
affected runtime content.

## Downstream Publication Boundary

Luas does not choose a registry, cloud identity, release environment, or trust root. A downstream
publisher should:

1. Build once and push by digest with registry-supported maximal provenance and SBOM attestations.
2. Sign the immutable registry digest through Cosign keyless OIDC or an organization-owned key.
3. Verify issuer, subject/workflow identity, repository, and digest at deployment admission.
4. Deploy the digest, not a mutable tag; keep the tag only as a human release pointer.
5. Retain evidence according to the downstream incident-response and compliance policy.

Maximal provenance records build arguments and invocation details. Never pass credentials through
`ARG` or ordinary build environment values. Use BuildKit secret/SSH mounts for build-only secrets and
runtime secret injection for application credentials.

## Update Procedure

1. Review official release notes and the exact multi-platform image index or scanner assets.
2. Update readable versions and digests together; never replace a digest with a movable tag.
3. Update verifier material expectations and `check-container-supply-chain.py` in the same change.
4. Build and run both affected images, export their SBOMs, and enforce the live scan.
5. Run `make governance` and `make check`.
6. Push the exact commit, inspect both workflow logs and artifacts, and confirm no warning was hidden
   behind a successful status.
