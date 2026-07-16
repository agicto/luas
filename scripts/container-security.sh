#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TRIVY_VERSION="0.72.0"
TRIVY_RELEASE_URL="https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_VERSION}"
DEFAULT_CACHE_HOME="${XDG_CACHE_HOME:-${HOME:-${TMPDIR:-/tmp}}/.cache}"
CACHE_ROOT="${LUAS_TRIVY_TOOL_CACHE_DIR:-${DEFAULT_CACHE_HOME}/luas/trivy/v${TRIVY_VERSION}}"
DB_CACHE="${TRIVY_CACHE_DIR:-${DEFAULT_CACHE_HOME}/luas/trivy/db}"

usage() {
  cat <<'EOF'
Usage: scripts/container-security.sh scan <image>
       scripts/container-security.sh sbom <image> [output.cdx.json]
       scripts/container-security.sh verify <image> [output.cdx.json]

scan    Fail on HIGH/CRITICAL vulnerabilities, secrets, or an EOL base OS.
sbom    Export and validate a CycloneDX 1.7 image inventory.
verify  Export the SBOM, then enforce the image scan policy.
EOF
}

resolve_asset() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"

  case "${os}:${arch}" in
    Darwin:x86_64)
      ASSET="trivy_${TRIVY_VERSION}_macOS-64bit.tar.gz"
      EXPECTED_SHA256="ee5e60df8a98e5b89fd74a6d86f9e5c7e9a266a35002cb1e43291698b3bfee08"
      BINARY_NAME="trivy"
      ;;
    Darwin:arm64|Darwin:aarch64)
      ASSET="trivy_${TRIVY_VERSION}_macOS-ARM64.tar.gz"
      EXPECTED_SHA256="88f208680dc05da2b459e19b4f5aa2b4dc7c2117892ba4aab2ae63baba330016"
      BINARY_NAME="trivy"
      ;;
    Linux:x86_64|Linux:amd64)
      ASSET="trivy_${TRIVY_VERSION}_Linux-64bit.tar.gz"
      EXPECTED_SHA256="bbb64b9695866ce4a7a8f5c9592002c5961cab378577fa3f8a040df362b9b2ea"
      BINARY_NAME="trivy"
      ;;
    Linux:aarch64|Linux:arm64)
      ASSET="trivy_${TRIVY_VERSION}_Linux-ARM64.tar.gz"
      EXPECTED_SHA256="2ca2c023109c2db6b2b77366b6717291452d4531167377d95c79547f0c8e3467"
      BINARY_NAME="trivy"
      ;;
    MINGW*:x86_64|MSYS*:x86_64|CYGWIN*:x86_64)
      ASSET="trivy_${TRIVY_VERSION}_windows-64bit.zip"
      EXPECTED_SHA256="ed3cf122060f61818fe1f735fd97557954e16e10bc8b058af9852271cf2e91b3"
      BINARY_NAME="trivy.exe"
      ;;
    *)
      echo "Unsupported Trivy platform: ${os}/${arch}" >&2
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

download_archive() {
  local archive="$1" download
  if [[ -f "${archive}" ]] && [[ "$(sha256_file "${archive}")" == "${EXPECTED_SHA256}" ]]; then
    return
  fi

  command -v curl >/dev/null 2>&1 || {
    echo "curl is required to download Trivy." >&2
    exit 2
  }

  mkdir -p "${CACHE_ROOT}"
  download="$(mktemp "${CACHE_ROOT}/.download.XXXXXX")"
  trap 'rm -f "${download:-}"' EXIT
  curl --proto '=https' --proto-redir '=https' --tlsv1.2 --fail --location --retry 3 \
    --connect-timeout 15 --max-time 300 --silent --show-error \
    "${TRIVY_RELEASE_URL}/${ASSET}" --output "${download}"

  if [[ "$(sha256_file "${download}")" != "${EXPECTED_SHA256}" ]]; then
    echo "Trivy archive checksum verification failed for ${ASSET}." >&2
    exit 2
  fi

  mv -f "${download}" "${archive}"
  trap - EXIT
}

install_scanner() {
  local archive extract_dir staged
  resolve_asset
  mkdir -p "${CACHE_ROOT}" "${DB_CACHE}"
  archive="${CACHE_ROOT}/${ASSET}"
  download_archive "${archive}"

  # Re-extract from the checksum-verified archive every run. A modified cached
  # executable is never trusted merely because it already exists.
  extract_dir="$(mktemp -d "${CACHE_ROOT}/.extract.XXXXXX")"
  staged="${extract_dir}/${BINARY_NAME}.staged"
  trap 'rm -rf "${extract_dir:-}"; rm -f "${staged:-}"' EXIT
  case "${ASSET}" in
    *.tar.gz)
      tar -xzf "${archive}" -C "${extract_dir}" "${BINARY_NAME}"
      ;;
    *.zip)
      command -v unzip >/dev/null 2>&1 || {
        echo "unzip is required to install Trivy on Windows." >&2
        exit 2
      }
      unzip -qq "${archive}" "${BINARY_NAME}" -d "${extract_dir}"
      ;;
  esac
  install -m 0755 "${extract_dir}/${BINARY_NAME}" "${staged}"
  mv -f "${staged}" "${CACHE_ROOT}/${BINARY_NAME}"
  rm -rf "${extract_dir}"
  trap - EXIT

  TRIVY="${CACHE_ROOT}/${BINARY_NAME}"
  "${TRIVY}" --version | grep -q "Version: ${TRIVY_VERSION}" || {
    echo "Installed Trivy does not report version ${TRIVY_VERSION}." >&2
    exit 2
  }
}

validate_image() {
  local image="$1"
  [[ -n "${image}" ]] && [[ "${image}" != -* ]] || {
    echo "Image reference must be non-empty and must not begin with '-'." >&2
    exit 2
  }
}

scan_image() {
  local image="$1"
  "${TRIVY}" image \
    --cache-dir "${DB_CACHE}" \
    --skip-version-check \
    --no-progress \
    --timeout 10m \
    --detection-priority precise \
    --scanners vuln,secret \
    --severity HIGH,CRITICAL \
    --exit-code 1 \
    --exit-on-eol 1 \
    --ignorefile "${ROOT_DIR}/.trivyignore.yaml" \
    "${image}"
}

export_sbom() {
  local image="$1" output="$2" output_dir temporary
  command -v python3 >/dev/null 2>&1 || {
    echo "python3 is required to validate container SBOMs." >&2
    exit 2
  }
  output_dir="$(dirname "${output}")"
  mkdir -p "${output_dir}"
  temporary="$(mktemp "${output_dir}/.luas-container-sbom.XXXXXX")"
  trap 'rm -f "${temporary:-}"' EXIT

  "${TRIVY}" image \
    --cache-dir "${DB_CACHE}" \
    --skip-version-check \
    --no-progress \
    --timeout 10m \
    --scanners vuln \
    --format cyclonedx \
    --output "${temporary}" \
    "${image}"

  python3 "${ROOT_DIR}/scripts/validate-container-sbom.py" "${temporary}" "${image}"

  mv -f "${temporary}" "${output}"
  trap - EXIT
  echo "Container SBOM: ${output}"
  echo "SHA-256: $(sha256_file "${output}")"
}

if (( $# == 1 )) && [[ "$1" =~ ^(-h|--help|help)$ ]]; then
  usage
  exit 0
fi

if (( $# < 2 )); then
  usage >&2
  exit 2
fi

command="$1"
image="$2"
shift 2
validate_image "${image}"
install_scanner

case "${command}" in
  scan)
    (( $# == 0 )) || { usage >&2; exit 2; }
    scan_image "${image}"
    ;;
  sbom)
    (( $# <= 1 )) || { usage >&2; exit 2; }
    export_sbom "${image}" "${1:-${TMPDIR:-/tmp}/luas-container.cdx.json}"
    ;;
  verify)
    (( $# <= 1 )) || { usage >&2; exit 2; }
    export_sbom "${image}" "${1:-${TMPDIR:-/tmp}/luas-container.cdx.json}"
    scan_image "${image}"
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
