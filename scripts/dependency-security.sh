#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OSV_VERSION="2.3.8"
OSV_RELEASE_URL="https://github.com/google/osv-scanner/releases/download/v${OSV_VERSION}"
DEFAULT_CACHE_HOME="${XDG_CACHE_HOME:-${HOME:-${TMPDIR:-/tmp}}/.cache}"
CACHE_ROOT="${OSV_SCANNER_CACHE_DIR:-${DEFAULT_CACHE_HOME}/luas/osv-scanner/v${OSV_VERSION}}"

usage() {
  cat <<'EOF'
Usage: scripts/dependency-security.sh scan
       scripts/dependency-security.sh sbom [output.cdx.json]

scan  Check api/go.mod and both browser lockfiles against the OSV database.
sbom  Export a validated CycloneDX 1.5 inventory without suppressing scan policy.
EOF
}

resolve_asset() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"

  case "${os}:${arch}" in
    Darwin:x86_64)
      ASSET="osv-scanner_darwin_amd64"
      EXPECTED_SHA256="b8a80a9f14ca4c0cd0fc2d351b28f740da9e6a5b18385ac9f9d083360b5b504e"
      ;;
    Darwin:arm64|Darwin:aarch64)
      ASSET="osv-scanner_darwin_arm64"
      EXPECTED_SHA256="a8cd6507b06239f463a7642430cfd2d154882f150f6e30cdc0653e28dfc34216"
      ;;
    Linux:x86_64|Linux:amd64)
      ASSET="osv-scanner_linux_amd64"
      EXPECTED_SHA256="bc98e15319ed0d515e3f9235287ba53cdc5535d576d24fd573978ecfe9ab92dc"
      ;;
    Linux:aarch64|Linux:arm64)
      ASSET="osv-scanner_linux_arm64"
      EXPECTED_SHA256="8158b18edd2d03b1a30d905ca91b032bc62262167be8f206c27114f08823e27c"
      ;;
    MINGW*:x86_64|MSYS*:x86_64|CYGWIN*:x86_64)
      ASSET="osv-scanner_windows_amd64.exe"
      EXPECTED_SHA256="cb04e79dd9698a7bc821bbfdddec916a416d1409fda79c927c509d37d00c9716"
      ;;
    MINGW*:arm64|MINGW*:aarch64|MSYS*:arm64|MSYS*:aarch64|CYGWIN*:arm64|CYGWIN*:aarch64)
      ASSET="osv-scanner_windows_arm64.exe"
      EXPECTED_SHA256="285d1fbcf2c69ab5ee38ae3a850ab46e83f32ef1cd5f3c4c9eb161cc493f6d52"
      ;;
    *)
      echo "Unsupported OSV-Scanner platform: ${os}/${arch}" >&2
      exit 2
      ;;
  esac
}

sha256_file() {
  local path="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${path}" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${path}" | awk '{print $1}'
  else
    echo "A SHA-256 utility (sha256sum or shasum) is required." >&2
    exit 2
  fi
}

verify_binary() {
  local path="$1"
  [[ -f "${path}" ]] && [[ "$(sha256_file "${path}")" == "${EXPECTED_SHA256}" ]]
}

install_scanner() {
  local binary download
  resolve_asset
  binary="${CACHE_ROOT}/${ASSET}"

  if ! verify_binary "${binary}"; then
    command -v curl >/dev/null 2>&1 || {
      echo "curl is required to download OSV-Scanner." >&2
      exit 2
    }

    mkdir -p "${CACHE_ROOT}"
    download="$(mktemp "${CACHE_ROOT}/.download.XXXXXX")"
    trap 'rm -f "${download:-}"' EXIT
    curl --proto '=https' --proto-redir '=https' --tlsv1.2 --fail --location --retry 3 \
      --connect-timeout 15 --max-time 180 \
      --silent --show-error "${OSV_RELEASE_URL}/${ASSET}" --output "${download}"

    if [[ "$(sha256_file "${download}")" != "${EXPECTED_SHA256}" ]]; then
      echo "OSV-Scanner checksum verification failed for ${ASSET}." >&2
      exit 2
    fi

    chmod 0755 "${download}"
    mv -f "${download}" "${binary}"
    trap - EXIT
  fi

  OSV_SCANNER="${binary}"
}

scanner_args() {
  SCAN_ARGS=(
    scan source
    "--config=${ROOT_DIR}/osv-scanner.toml"
    "--lockfile=${ROOT_DIR}/api/go.mod"
    "--lockfile=${ROOT_DIR}/web/pnpm-lock.yaml"
    "--lockfile=${ROOT_DIR}/web-spa/pnpm-lock.yaml"
    --verbosity=warn
  )
}

scan_dependencies() {
  "${OSV_SCANNER}" "${SCAN_ARGS[@]}"
}

export_sbom() {
  local output="$1" output_dir temporary status
  output_dir="$(dirname "${output}")"
  mkdir -p "${output_dir}"
  temporary="$(mktemp "${output_dir}/.luas-sbom.XXXXXX")"
  trap 'rm -f "${temporary:-}"' EXIT

  set +e
  "${OSV_SCANNER}" "${SCAN_ARGS[@]}" \
    --format=cyclonedx-1-5 \
    --all-packages \
    --output-file="${temporary}"
  status=$?
  set -e

  if (( status > 1 )); then
    echo "OSV-Scanner could not generate the SBOM (exit ${status})." >&2
    exit "${status}"
  fi

  python3 - "${temporary}" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, encoding="utf-8") as handle:
    document = json.load(handle)

if document.get("bomFormat") != "CycloneDX" or document.get("specVersion") != "1.5":
    raise SystemExit("OSV-Scanner did not produce a CycloneDX 1.5 document")

components = document.get("components")
if not isinstance(components, list) or not components:
    raise SystemExit("CycloneDX document contains no components")

purls = {component.get("purl", "") for component in components}
if not any(purl.startswith("pkg:golang/") for purl in purls):
    raise SystemExit("CycloneDX document is missing Go modules")
if not any(purl.startswith("pkg:npm/") for purl in purls):
    raise SystemExit("CycloneDX document is missing npm packages")

print(f"Validated CycloneDX 1.5 SBOM with {len(components)} components.")
PY

  mv -f "${temporary}" "${output}"
  trap - EXIT
  echo "SBOM: ${output}"
  echo "SHA-256: $(sha256_file "${output}")"
}

if (( $# < 1 )); then
  usage >&2
  exit 2
fi

command="$1"
shift

case "${command}" in
  scan)
    (( $# == 0 )) || { usage >&2; exit 2; }
    install_scanner
    scanner_args
    scan_dependencies
    ;;
  sbom)
    (( $# <= 1 )) || { usage >&2; exit 2; }
    install_scanner
    scanner_args
    export_sbom "${1:-${TMPDIR:-/tmp}/luas.cdx.json}"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
