#!/usr/bin/env bash
set -euo pipefail

PURGE=false
case "$#" in
  0) ;;
  1)
    if [[ "$1" == "--purge" ]]; then
      PURGE=true
    else
      echo "usage: $0 [--purge]" >&2
      exit 2
    fi
    ;;
  *)
    echo "usage: $0 [--purge]" >&2
    exit 2
    ;;
esac

if [[ "${EUID}" -ne 0 ]]; then
  echo "error: uninstall.sh must be run as root" >&2
  exit 1
fi

command -v systemctl >/dev/null || { echo "error: systemctl is required" >&2; exit 1; }
systemctl disable --now opspilot-agent.service >/dev/null 2>&1 || true
rm -f -- /usr/lib/systemd/system/opspilot-agent.service /lib/systemd/system/opspilot-agent.service
systemctl daemon-reload
systemctl reset-failed opspilot-agent.service >/dev/null 2>&1 || true
rm -f -- /usr/local/bin/opspilot-agent

if [[ "${PURGE}" == true ]]; then
  rm -rf -- /etc/opspilot-agent /var/lib/opspilot-agent
  if id opspilot-agent >/dev/null 2>&1; then
    userdel opspilot-agent
  fi
  if getent group opspilot-agent >/dev/null 2>&1; then
    groupdel opspilot-agent
  fi
  echo "OpsPilot Agent uninstalled; configuration, state, user, and group purged."
else
  echo "OpsPilot Agent uninstalled; /etc/opspilot-agent and /var/lib/opspilot-agent were preserved."
fi
