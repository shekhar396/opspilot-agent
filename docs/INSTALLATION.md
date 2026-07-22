# Installing OpsPilot Agent on Linux

OpsPilot Agent supports systemd-based Linux distributions on `amd64` and `arm64`. The project remains early-stage and is not production-ready. Installation does not configure authentication, registration, or an OpsPilot server.

## Release archive

Each release archive has a versioned top-level directory containing the agent binary, license, README, example configuration, systemd unit, install and uninstall scripts, and this guide. Download the archive for your architecture together with `checksums.txt`.

Verify all downloaded archives listed in the checksum file:

```bash
sha256sum -c checksums.txt
```

If only one archive was downloaded, select its checksum line first:

```bash
grep 'opspilot-agent_0.1.0_linux_amd64.tar.gz$' checksums.txt | sha256sum -c -
```

Extract and install:

```bash
tar -xzf opspilot-agent_0.1.0_linux_amd64.tar.gz
cd opspilot-agent_0.1.0_linux_amd64
sudo ./scripts/install.sh
```

The installer requires root, creates a locked `opspilot-agent` system account, installs the files, reloads systemd, and enables the service. It does not start the service, preventing an immediate connection attempt with the example URL.

## Filesystem layout

| Path | Purpose | Ownership/mode |
| --- | --- | --- |
| `/usr/local/bin/opspilot-agent` | Agent executable | `root:root`, `0755` |
| `/etc/opspilot-agent/config.yaml` | Configuration | `root:opspilot-agent`, `0640` |
| `/var/lib/opspilot-agent` | Identity and state directory | `opspilot-agent:opspilot-agent`, `0700` |
| `/usr/lib/systemd/system/opspilot-agent.service` or `/lib/systemd/system/opspilot-agent.service` | Service unit | `root:root`, `0644` |

The installer prefers `/usr/lib/systemd/system` when present and otherwise uses `/lib/systemd/system`.

## Configure and start

Edit `/etc/opspilot-agent/config.yaml`. Set a real agent name, a valid controlled HTTPS server URL, and keep the identity path at `/var/lib/opspilot-agent/agent-id`. The example endpoint is a placeholder.

Validate before starting:

```bash
sudo -u opspilot-agent \
  /usr/local/bin/opspilot-agent validate-config \
  --config /etc/opspilot-agent/config.yaml
```

Start and inspect the service:

```bash
sudo systemctl start opspilot-agent
sudo systemctl status opspilot-agent
sudo journalctl -u opspilot-agent -f
```

Routine lifecycle commands are:

```bash
sudo systemctl restart opspilot-agent
sudo systemctl stop opspilot-agent
```

The identity file is created on the first successful runtime start with mode `0600`. It is reused across restarts, upgrades, ordinary uninstallations, and reinstallations. Heartbeat sequences are process-local and restart at 1 for every process.

## Upgrade

1. Download the new archive and verify its checksum.
2. Extract it.
3. Optionally stop the service.
4. Run the new archive's `sudo ./scripts/install.sh`.
5. Confirm that the existing configuration and identity remain.
6. Restart the service and verify its version and journal logs.

Repeated installation replaces the binary and unit, normalizes secure permissions, and preserves configuration contents and state. It never starts a stopped service unexpectedly.

## Uninstall

Remove the binary and unit while preserving configuration, state, identity, user, and group:

```bash
sudo ./scripts/uninstall.sh
```

Remove those retained files and the dedicated account as well:

```bash
sudo ./scripts/uninstall.sh --purge
```

Purge is destructive and cannot restore the persistent identity.

## Service hardening

The unit runs as the unprivileged service user, grants no Linux capabilities, uses a restrictive umask, and enables systemd protections for the host filesystem, home directories, kernel controls, privilege escalation, executable memory, real-time scheduling, and set-user-ID behavior. Only `/var/lib/opspilot-agent` is writable. Outbound HTTPS remains available.

## Troubleshooting

- Configuration failure: run `validate-config` as the service user and correct the reported field.
- Permission failure: verify the ownership and modes in the filesystem table.
- Service failure: inspect `systemctl status opspilot-agent` and `journalctl -u opspilot-agent -n 100 --no-pager`.
- Connection failure: verify DNS, outbound HTTPS, the configured base URL, and server trust. Do not disable TLS verification.
- Redirect rejection: automatic redirects are intentionally disabled to prevent forwarding identity headers.
- Installation leaves the service inactive: this is intentional; configure it and start it explicitly.

The agent has no retries or offline queue. A failed heartbeat waits for the next normal interval.
