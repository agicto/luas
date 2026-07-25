# Dependency Supply-Chain Security

Luas treats dependency resolution as a repository-wide operational boundary. The API, Next.js Web,
and static SPA keep independent module systems, while one root workflow inventories and scans all
three lock surfaces.

## Control Model

| Control | Repository authority | Guarantee |
|---|---|---|
| Browser runtimes | `web/package.json`, `web-spa/package.json` | Only maintained Node 22.12+/24 LTS lines are accepted; Node types stay at the deployed Node 22 baseline. |
| Browser package manager | Both browser `package.json` files | pnpm is pinned to exact version `10.34.5`; mismatched pnpm and competing lockfiles fail governance. |
| Browser resolution policy | Both `pnpm-workspace.yaml` files | New versions wait 24 hours, recent trust evidence cannot downgrade, and transitive exotic sources are blocked. |
| Dependency scripts | Both `pnpm-workspace.yaml` files | Unreviewed install scripts fail; only five exact native/build package versions may execute them. |
| Locked content | `web/pnpm-lock.yaml`, `web-spa/pnpm-lock.yaml` | Every registry package carries integrity evidence and frozen installs are required in CI. |
| Vulnerability source | `scripts/dependency-security.sh` | OSV-Scanner 2.3.8 binaries are selected per platform and verified against reviewed SHA-256 digests. |
| Inventory | `make sbom` | A validated CycloneDX 1.5 document contains both Go modules and npm packages. |
| Continuous review | `.github/workflows/dependency-security.yml` | Dependency changes, weekly schedules, and manual runs scan both lock surfaces and retain the SBOM for 14 days. |
| Update discovery | `.github/dependabot.yml` | Weekly grouped updates cover Go, both pnpm projects, GitHub Actions, and both Dockerfiles; major updates remain separate review units. |

Node 20 is intentionally absent because it is end-of-life and no longer receives security fixes.
Node 22.12 is the minimum browser-tooling runtime and Node 22 remains the image/type-definition
baseline; CI verifies both Node 22 and Node 24. The 90-day
`trustPolicyIgnoreAfter` window is deliberate. Newer releases must not downgrade known
registry trust evidence; old packages without historical provenance remain installable and are
still covered by integrity checks and vulnerability scanning. The 24-hour release-age rule applies
when pnpm resolves updates, including transitive packages. It does not replace review of lockfile
diffs.

## Commands

```bash
# Network-backed vulnerability gate for api/go.mod and both browser lockfiles
make dependency-scan

# Write a CycloneDX 1.5 inventory outside the repository by default
make sbom

# Choose an explicit artifact path
SBOM_OUTPUT="$PWD/luas.cdx.json" make sbom

# Deterministic policy checks only; no vulnerability-network dependency
make governance
```

`make check` intentionally keeps the live advisory lookup separate. Builds and local correctness
must not become flaky when an external vulnerability service is unavailable; the dedicated workflow
and `make dependency-scan` own that network-backed gate.

OSV-Scanner sends package names, versions, ecosystems, and file hashes to OSV/deps.dev as needed; it
does not upload repository source. Generated SBOMs are build artifacts, not committed source. Review
their distribution like any dependency inventory, especially for private downstream applications.
Source-lock scanning does not inspect operating-system packages or the final runtime filesystem. See
[`CONTAINER_SECURITY.md`](CONTAINER_SECURITY.md) for digest-pinned images, BuildKit evidence,
image-level CycloneDX inventory, and the separate Trivy gate.

## Exceptions

The default [`../osv-scanner.toml`](../osv-scanner.toml) contains no suppression. A temporary
`[[IgnoredVulns]]` entry must include:

- the exact advisory `id`;
- a non-empty, reachability-specific `reason`;
- a non-expired `ignoreUntil` date.

Any `[[PackageOverrides]]` entry likewise requires `reason` and `effectiveUntil`. Root governance
fails expired or incomplete exceptions. Do not suppress a package family, severity class, or scanner
failure. Prefer upgrading, replacing, or removing the dependency; record an exception only when the
advisory is proven unreachable and remediation cannot land immediately.

## License Boundary

Luas inventories dependencies but does not impose a universal commercial license allowlist. License
compatibility depends on downstream distribution, linking, and legal policy. Review production
licenses with `corepack pnpm licenses list --prod` in each retained browser shell, inspect the SBOM,
and apply an organization-owned policy before release. A scanner allowlist must not be presented as
legal advice.

## Updating The Toolchain

1. Review the official pnpm or OSV release notes and signatures.
2. Keep Node on maintained LTS lines and its types at the lowest deployed runtime baseline.
3. Keep pnpm on a Dependabot-supported major and update `packageManager`, `engines.pnpm`, and the
   governance constant together.
4. For OSV-Scanner, resolve every supported release asset and digest from the official release, then
   update the script and governance constants together.
5. Run a mismatched-pnpm negative test, a clean frozen install, `make dependency-scan`, `make sbom`,
   and `make check` (`make check` includes governance).
6. Push the exact commit and inspect the Dependency Security workflow artifact and annotations.

Primary references: [pnpm 10 settings](https://pnpm.io/10.x/settings),
[OSV supported lockfiles](https://google.github.io/osv-scanner/supported-languages-and-lockfiles/),
[OSV output formats](https://google.github.io/osv-scanner/output/), and
[Dependabot configuration](https://docs.github.com/en/code-security/reference/supply-chain-security/dependabot-options-reference).
