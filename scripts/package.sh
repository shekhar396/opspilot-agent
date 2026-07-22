#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
VERSION="${VERSION:-}"
COMMIT="${COMMIT:-}"
BUILD_DATE="${BUILD_DATE:-}"

for name in VERSION COMMIT BUILD_DATE; do
  if [[ -z "${!name}" ]]; then
    echo "error: ${name} is required" >&2
    exit 1
  fi
done

ARCHIVE_VERSION="${VERSION#v}"
if [[ ! "${ARCHIVE_VERSION}" =~ ^[0-9A-Za-z][0-9A-Za-z._+-]*$ ]]; then
  echo "error: VERSION is not safe for an archive filename" >&2
  exit 1
fi

DIST_DIR="${PROJECT_ROOT}/dist"
STAGING_DIR="$(mktemp -d)"
trap 'rm -rf -- "${STAGING_DIR}"' EXIT
rm -rf -- "${DIST_DIR}"
mkdir -p -- "${DIST_DIR}"

MODULE="github.com/shekhar396/opspilot-agent"
LDFLAGS="-X ${MODULE}/internal/version.Version=${VERSION} -X ${MODULE}/internal/version.Commit=${COMMIT} -X ${MODULE}/internal/version.Date=${BUILD_DATE}"

for arch in amd64 arm64; do
  root_name="opspilot-agent_${ARCHIVE_VERSION}_linux_${arch}"
  root_dir="${STAGING_DIR}/${root_name}"
  mkdir -p -- "${root_dir}/configs" "${root_dir}/deployments/systemd" "${root_dir}/scripts" "${root_dir}/docs"

  CGO_ENABLED=0 GOOS=linux GOARCH="${arch}" go build -trimpath -ldflags "${LDFLAGS}" \
    -o "${root_dir}/opspilot-agent" "${PROJECT_ROOT}/cmd/opspilot-agent"
  chmod 0755 "${root_dir}/opspilot-agent"
  install -m 0644 "${PROJECT_ROOT}/LICENSE" "${root_dir}/LICENSE"
  install -m 0644 "${PROJECT_ROOT}/README.md" "${root_dir}/README.md"
  install -m 0644 "${PROJECT_ROOT}/configs/opspilot-agent.example.yaml" "${root_dir}/configs/opspilot-agent.example.yaml"
  install -m 0644 "${PROJECT_ROOT}/deployments/systemd/opspilot-agent.service" "${root_dir}/deployments/systemd/opspilot-agent.service"
  install -m 0755 "${PROJECT_ROOT}/scripts/install.sh" "${root_dir}/scripts/install.sh"
  install -m 0755 "${PROJECT_ROOT}/scripts/uninstall.sh" "${root_dir}/scripts/uninstall.sh"
  install -m 0644 "${PROJECT_ROOT}/docs/INSTALLATION.md" "${root_dir}/docs/INSTALLATION.md"

  tar --sort=name --mtime='UTC 1970-01-01' --owner=0 --group=0 --numeric-owner \
    -C "${STAGING_DIR}" -czf "${DIST_DIR}/${root_name}.tar.gz" "${root_name}"
done

(
  cd -- "${DIST_DIR}"
  sha256sum ./*.tar.gz | LC_ALL=C sort -k2 > checksums.txt
)
