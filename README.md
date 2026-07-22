# OpsPilot Agent

> The project is in early development and is not production-ready.

## Overview

OpsPilot Agent is a small, installable Linux heartbeat agent. Phase 1 is complete after Step 10, and the repository is prepared for an initial tagged release. Evidence collection, registration, authentication, and action execution are not implemented.

> OpsPilot Agent is a lightweight Linux operations agent. It collects approved operational evidence and communicates with OpsPilot AI. AI reasoning does not run inside the agent.

## What OpsPilot Agent Is

OpsPilot Agent is intended to become:

- A lightweight Linux host agent.
- A collector of explicitly approved operational evidence.
- A securely communicating component of the OpsPilot ecosystem.
- A controlled executor of predefined allow-listed actions in later milestones.

These broader capabilities are planned and are not implemented in Phase 1.

## What OpsPilot Agent Is Not

OpsPilot Agent is not:

- An AI model.
- An autonomous AI agent.
- A general-purpose remote shell.
- A replacement for Prometheus.
- A full log-management platform.
- A Kubernetes agent at this stage.

## Current Scope

Phase 1 currently provides:

- Initial Go module and public repository structure.
- Cobra-based CLI foundation.
- Explicit command constructors.
- Minimal agent runtime skeleton.
- Graceful shutdown on SIGINT and SIGTERM.
- Structured logging using the standard library `log/slog` package.
- JSON and text log formats with debug, info, warn, and error levels.
- Runtime startup and shutdown logs.
- Explicit logger injection into the runtime.
- Random persistent local agent identity.
- Secure identity-file creation and reuse.
- Versioned heartbeat payload construction and validation.
- Strict heartbeat JSON encoding and decoding.
- On-demand HTTP transport for one prebuilt heartbeat payload.
- Immediate and scheduled heartbeat delivery from the runtime.
- Non-fatal heartbeat rejection, timeout, and network-error handling.
- A production HTTP client that does not follow redirects.
- Context cancellation, request deadlines, and bounded response handling.
- Build-injectable version metadata.
- Strict YAML configuration loading.
- Configuration default values and validation.
- A real `validate-config` command.
- An example configuration file.
- Capability reporting for currently implemented capabilities.
- Static Linux release builds for `amd64` and `arm64`.
- Versioned release archives and SHA-256 checksums.
- An unprivileged, hardened systemd service.
- Idempotent installation and explicit purge tooling.
- CI and tag-driven GitHub Release workflows.

## Planned Architecture

```text
Linux Server
    |
    └── OpsPilot Agent
              |
              | HTTPS
              v
         OpsPilot AI
              |
              v
       Human Operator
```

Registration, collectors, and controlled actions are future milestones and are not implemented in Phase 1.

## Requirements

- Go installed locally
- Git
- Linux, macOS, or Windows for initial development
- Linux as the initial runtime target

## Build

```bash
make build
```

Common local development commands are:

```bash
make help
make fmt
make test
make test-race
make vet
make check
```

Create a release-style binary with explicit metadata:

```bash
make build-release VERSION=v0.1.0 COMMIT=abc1234 \
  BUILD_DATE=2026-07-22T12:00:00Z GOOS=linux GOARCH=amd64
```

`make package` creates versioned `linux/amd64` and `linux/arm64` archives plus `dist/checksums.txt`. Release builds use `CGO_ENABLED=0`, `-trimpath`, and linker-injected version metadata.

## Installation and Releases

Release archives contain the executable, example configuration, systemd unit, installer, uninstaller, license, README, and installation guide. Verify the SHA-256 checksum, extract the matching architecture, and run:

```bash
sudo ./scripts/install.sh
```

The installer creates an unprivileged `opspilot-agent` service account and uses these paths:

```text
/usr/local/bin/opspilot-agent
/etc/opspilot-agent/config.yaml
/var/lib/opspilot-agent/agent-id
/usr/lib/systemd/system/opspilot-agent.service
```

On systems without `/usr/lib/systemd/system`, `/lib/systemd/system` is used. The service is enabled but deliberately not started until the operator edits and validates the configuration. See [Installation](docs/INSTALLATION.md) for installation, upgrade, logs, and uninstall instructions, and [Release Process](docs/RELEASE.md) for the maintainer workflow.

CI verifies formatting, tests, race tests, vet, builds, configuration, capabilities, version output, and shell syntax. Tags matching `v*` trigger Linux packaging and GitHub Release creation. No release is claimed to exist until a reviewed tag is published.

## Run

```bash
./bin/opspilot-agent
```

The root command displays help and exits successfully.

## Configuration

The current configuration schema is intentionally small:

```yaml
agent:
  name: app-server-01
  server_url: https://opspilot.example.com
  heartbeat_interval: 30s
  request_timeout: 10s
  identity_file: /var/lib/opspilot-agent/agent-id

logging:
  level: info
  format: json
```

Unknown fields and multiple YAML documents are rejected. `agent.name` accepts only letters, numbers, periods, underscores, and hyphens, with a maximum length of 128 characters. `agent.server_url` must be an HTTPS URL without credentials, query parameters, or fragments; an optional base path is preserved for heartbeat requests. `agent.heartbeat_interval` must be between `5s` and `1h`. `agent.request_timeout` defaults to `10s`, must be between `100ms` and `2m`, and must be shorter than the heartbeat interval.

Supported logging levels are `debug`, `info`, `warn`, and `error`. Supported logging formats are `json` and `text`. These values are case-sensitive, and the current schema does not support secrets.

Create a local configuration from the tracked example and validate it:

```bash
cp configs/opspilot-agent.example.yaml configs/opspilot-agent.yaml
go run ./cmd/opspilot-agent validate-config
```

The local `configs/opspilot-agent.yaml` path is ignored by Git. The example can also be validated directly:

```bash
go run ./cmd/opspilot-agent validate-config \
  --config configs/opspilot-agent.example.yaml
```

Configuration is loaded and validated when the runtime starts.

## Persistent Agent Identity

The agent creates a random UUIDv4-compatible local identity and stores it at `agent.identity_file`. The default path is:

```text
/var/lib/opspilot-agent/agent-id
```

The ID persists across restarts, and existing valid identity files are reused. Malformed identity files cause startup to fail and are never silently replaced. Parent directories created by the agent use mode `0700`, and identity files use mode `0600` on Linux.

The identity is generated with cryptographically secure randomness. It is not derived from hardware, MAC or IP addresses, hostname, machine ID, user information, or configuration values. This is only a local persistent identity; server registration and authentication are not implemented.

Regular non-root development users should choose a writable local path. Do not use `/tmp` for production. For example:

```bash
mkdir -p configs
cp configs/opspilot-agent.example.yaml configs/opspilot-agent.yaml
```

Then edit the local file:

```yaml
agent:
  identity_file: /tmp/opspilot-agent-dev/agent-id
```

Run the agent:

```bash
go run ./cmd/opspilot-agent run
```

Startup and shutdown logs include the persisted `agent_id`.

## Logging

Logging is configured in YAML:

```yaml
logging:
  level: info
  format: json
```

Allowed levels are `debug`, `info`, `warn`, and `error`. Allowed formats are `json` and `text`. JSON and info are the defaults, and unsupported values are rejected during configuration validation.

Runtime logs are written to standard output. CLI and startup errors remain on standard error. The logger does not write files; log rotation is expected to be managed later by the operating system or service manager. Sensitive values must not be logged.

Start the runtime with a local configuration:

```bash
cp configs/opspilot-agent.example.yaml configs/opspilot-agent.yaml
# Edit agent.identity_file to a writable path for local development.
go run ./cmd/opspilot-agent run
```

It emits a startup log, waits for SIGINT or SIGTERM, and emits a shutdown log. The following is an illustrative JSON entry; timestamps and key ordering are not fixed:

```json
{
  "time": "2026-07-22T12:00:00Z",
  "level": "INFO",
  "msg": "agent runtime started",
  "agent_id": "9fb42f1c-8a12-4db5-a42c-7a4be50efaf1",
  "agent_name": "app-server-01",
  "server_url": "https://opspilot.example.com"
}
```

## Heartbeat Payload Foundation

Step 7 defines and validates the heartbeat protocol model. The following payload is illustrative; JSON key ordering is not guaranteed:

```json
{
  "schema_version": "1",
  "agent_id": "9fb42f1c-8a12-4db5-a42c-7a4be50efaf1",
  "agent_name": "app-server-01",
  "agent_version": "dev",
  "sent_at": "2026-07-22T12:00:00Z",
  "sequence": 1
}
```

- `schema_version` is the protocol schema version, currently `"1"`.
- `agent_id` is the persistent local agent identity.
- `agent_name` is the configured human-readable agent name.
- `agent_version` is the running OpsPilot Agent version.
- `sent_at` is the UTC payload creation timestamp.
- `sequence` is a process-local heartbeat sequence beginning at 1.

Heartbeat construction and validation are implemented, along with compact JSON encoding and strict decoding. Strict decoding rejects unknown fields, missing values, invalid values, and additional JSON documents.

The runtime constructs this payload for each delivery attempt. Agent metrics and host data are not part of it. Future protocol expansion should be deliberate and schema-versioned.

## HTTP Transport Foundation

The transport package can send one already-constructed heartbeat payload using an HTTP `POST` to:

```text
/api/v1/agent/heartbeat
```

A base URL path supplied to the transport is preserved when constructing the endpoint. The transport requires HTTPS, uses an explicitly injected `http.Client`, and does not mutate that client. The CLI creates a dedicated production client that does not follow redirects, preventing agent identity headers from being forwarded to a redirect target.

Request cancellation and deadlines use the caller context. A per-request context timeout comes from `agent.request_timeout`, which defaults to `10s` and must remain shorter than `agent.heartbeat_interval`.

The initial successful status contract is deliberately limited to:

```text
200 OK
202 Accepted
204 No Content
```

Other statuses return a typed rejection error. Response-body reads are capped at 8 KiB, retained error messages are further bounded, and server-provided `Content-Length` does not control allocation. Server request IDs are read from `X-Request-ID`.

The runtime uses this transport for scheduled heartbeat delivery. There is no retry, backoff, registration, or authentication.

## Heartbeat Runtime

The runtime sends one heartbeat immediately after startup and then one after each configured `agent.heartbeat_interval`. Heartbeats are sent synchronously, so delivery attempts never overlap. Sequence numbers are process-local: they begin at 1, increment once per attempt, and reset when the process restarts. Every payload timestamp is freshly generated in UTC.

Successful delivery is logged at `INFO`. Server rejections are logged at `WARN`; network failures and request timeouts are logged at `ERROR`. These delivery failures do not stop the runtime. Logs include the agent ID and sequence, plus the HTTP status, request ID, or failure type when available. At higher configured log levels, lower-severity lifecycle and delivery entries are filtered normally.

Cancellation stops the schedule and cancels an in-flight request through its context. The runtime then shuts down cleanly. Failed heartbeats are not queued, and there are no immediate retries: the next attempt occurs only on the next configured interval. Sequence values are not persisted across restarts.

The lifecycle is conceptually:

```text
runtime started
    |
    +-- heartbeat delivered/rejected/failed
    |
    +-- next interval
    |
runtime stopped
```

Authentication, registration, server-issued commands, metrics, collectors, and host inspection remain absent.

## CLI Usage

```bash
go run ./cmd/opspilot-agent --help
go run ./cmd/opspilot-agent run --config configs/opspilot-agent.example.yaml
go run ./cmd/opspilot-agent version
go run ./cmd/opspilot-agent validate-config
go run ./cmd/opspilot-agent print-capabilities
```

Current command output:

```text
$ opspilot-agent run --config configs/opspilot-agent.example.yaml
{"time":"...","level":"INFO","msg":"agent runtime started","agent_id":"...","agent_name":"app-server-01","server_url":"https://opspilot.example.com"}
# Waits until SIGINT or SIGTERM, then emits the shutdown log.

$ opspilot-agent version
opspilot-agent version dev
commit: unknown
built: unknown

$ opspilot-agent validate-config --config configs/opspilot-agent.example.yaml
Configuration is valid

$ opspilot-agent print-capabilities
cli
version
config-validation
structured-logging
runtime
persistent-identity
heartbeat-payload
http-transport
heartbeat-runtime
linux-service
release-packaging
```

The `run` command loads validated configuration, loads or creates the persistent identity, creates the heartbeat transport and runtime, sends scheduled heartbeats, and waits for SIGINT or SIGTERM. The `validate-config` command validates a file without starting the runtime, and `print-capabilities` reports only implemented capabilities.

## Current Limitations

Heartbeat communication is deliberately minimal: failed attempts are not retried or queued, sequences are not persisted, and there are no delivery guarantees. No authentication, agent registration, offline queue, host telemetry, metrics, server-issued commands, or remote actions exist. Linux collection, systemd inspection, process monitoring, log shipping, Docker, and Kubernetes support also remain unimplemented.

## Roadmap

1. Repository and CLI foundation.
2. Configuration, structured logging, versioning, and graceful shutdown.
3. Persistent identity and heartbeat communication.
4. Linux host collection.
5. Additional controlled collectors and secure actions.

## License

Licensed under the Apache License 2.0. See LICENSE.
