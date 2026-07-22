#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 0 ]]; then
  echo "usage: $0" >&2
  exit 2
fi
if [[ "${EUID}" -ne 0 ]]; then
  echo "error: install.sh must be run as root" >&2
  exit 1
fi

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
if [[ -x "${PROJECT_ROOT}/opspilot-agent" ]]; then
  SOURCE_BINARY="${PROJECT_ROOT}/opspilot-agent"
elif [[ -x "${PROJECT_ROOT}/bin/opspilot-agent" ]]; then
  SOURCE_BINARY="${PROJECT_ROOT}/bin/opspilot-agent"
else
  echo "error: executable opspilot-agent binary not found in archive root or bin/" >&2
  exit 1
fi

SOURCE_CONFIG="${PROJECT_ROOT}/configs/opspilot-agent.example.yaml"
SOURCE_UNIT="${PROJECT_ROOT}/deployments/systemd/opspilot-agent.service"
[[ -f "${SOURCE_CONFIG}" ]] || { echo "error: example configuration not found" >&2; exit 1; }
[[ -f "${SOURCE_UNIT}" ]] || { echo "error: systemd unit not found" >&2; exit 1; }
command -v systemctl >/dev/null || { echo "error: systemctl is required" >&2; exit 1; }
command -v useradd >/dev/null || { echo "error: useradd is required" >&2; exit 1; }

if [[ -d /usr/lib/systemd/system ]]; then
  UNIT_DIR=/usr/lib/systemd/system
elif [[ -d /lib/systemd/system ]]; then
  UNIT_DIR=/lib/systemd/system
else
  echo "error: no supported systemd unit directory found" >&2
  exit 1
fi

if ! id opspilot-agent >/dev/null 2>&1; then
  if getent group opspilot-agent >/dev/null 2>&1; then
    useradd --system --home-dir /var/lib/opspilot-agent --shell /usr/sbin/nologin --gid opspilot-agent opspilot-agent
  else
    useradd --system --home-dir /var/lib/opspilot-agent --shell /usr/sbin/nologin --user-group opspilot-agent
  fi
fi
getent group opspilot-agent >/dev/null 2>&1 || { echo "error: opspilot-agent group is missing" >&2; exit 1; }

install -d -m 0755 -o root -g root /etc/opspilot-agent
install -d -m 0700 -o opspilot-agent -g opspilot-agent /var/lib/opspilot-agent
install -m 0755 -o root -g root "${SOURCE_BINARY}" /usr/local/bin/opspilot-agent
if [[ -e /etc/opspilot-agent/config.yaml ]]; then
  echo "Preserving existing /etc/opspilot-agent/config.yaml"
else
  install -m 0640 -o root -g opspilot-agent "${SOURCE_CONFIG}" /etc/opspilot-agent/config.yaml
  echo "Installed example configuration at /etc/opspilot-agent/config.yaml"
fi
chown root:opspilot-agent /etc/opspilot-agent/config.yaml
chmod 0640 /etc/opspilot-agent/config.yaml

install -m 0644 -o root -g root "${SOURCE_UNIT}" "${UNIT_DIR}/opspilot-agent.service"
systemctl daemon-reload
systemctl enable opspilot-agent.service

echo "Installation complete. The service was enabled but not started."
echo "Next steps:"
echo "  1. Edit /etc/opspilot-agent/config.yaml"
echo "  2. sudo -u opspilot-agent /usr/local/bin/opspilot-agent validate-config --config /etc/opspilot-agent/config.yaml"
echo "  3. systemctl start opspilot-agent"
echo "  4. systemctl status opspilot-agent"
echo "  5. journalctl -u opspilot-agent -f"
