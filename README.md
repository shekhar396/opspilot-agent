# OpsPilot Agent

> The project is in early development and is not production-ready.

## Overview

OpsPilot Agent is planned to run on Linux virtual machines and eventually communicate with OpsPilot AI. Runtime communication, evidence collection, registration, authentication, and action execution are not implemented in Step 8.

> OpsPilot Agent is a lightweight Linux operations agent. It collects approved operational evidence and communicates with OpsPilot AI. AI reasoning does not run inside the agent.

## What OpsPilot Agent Is

OpsPilot Agent is intended to become:

- A lightweight Linux host agent.
- A collector of explicitly approved operational evidence.
- A securely communicating component of the OpsPilot ecosystem.
- A controlled executor of predefined allow-listed actions in later milestones.

These capabilities are planned and are not implemented in Step 8.

## What OpsPilot Agent Is Not

OpsPilot Agent is not:

- An AI model.
- An autonomous AI agent.
- A general-purpose remote shell.
- A replacement for Prometheus.
- A full log-management platform.
- A Kubernetes agent at this stage.

## Current Scope

Step 8 currently provides:

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
- Context cancellation, request deadlines, and bounded response handling.
- Build-injectable version metadata.
- Strict YAML configuration loading.
- Configuration default values and validation.
- A real `validate-config` command.
- An example configuration file.
- Capability reporting for currently implemented capabilities.

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

Runtime heartbeat transmission and scheduling, registration, collectors, and controlled actions are future milestones and are not implemented in Step 8.

## Requirements

- Go installed locally
- Git
- Linux, macOS, or Windows for initial development
- Linux as the initial runtime target

## Build

```bash
go build -o bin/opspilot-agent ./cmd/opspilot-agent
```

Future release builds can inject version information:

```bash
go build \
  -ldflags "\
-X github.com/shekhar396/opspilot-agent/internal/version.Version=v0.1.0 \
-X github.com/shekhar396/opspilot-agent/internal/version.Commit=abc1234 \
-X github.com/shekhar396/opspilot-agent/internal/version.Date=2026-07-22T12:00:00Z" \
  -o bin/opspilot-agent \
  ./cmd/opspilot-agent
```

Then inspect the injected values with:

```bash
./bin/opspilot-agent version
```

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

Unknown fields and multiple YAML documents are rejected. `agent.name` accepts only letters, numbers, periods, underscores, and hyphens, with a maximum length of 128 characters. `agent.server_url` must be an HTTPS URL without credentials, query parameters, fragments, or a non-root path. `agent.heartbeat_interval` must be between `5s` and `1h`. `agent.request_timeout` defaults to `10s`, must be between `100ms` and `2m`, and must be shorter than the heartbeat interval.

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

The runtime transmits no heartbeat, and no timer or scheduler exists yet. Sequence persistence is not implemented. Agent metrics and host data are not part of this payload. Future protocol expansion should be deliberate and schema-versioned.

## HTTP Transport Foundation

The transport package can send one already-constructed heartbeat payload using an HTTP `POST` to:

```text
/api/v1/agent/heartbeat
```

A base URL path supplied to the transport is preserved when constructing the endpoint. The transport requires HTTPS, uses an explicitly injected `http.Client`, and does not mutate that client. Redirect behavior follows the injected client’s configuration; production callers should adopt a deliberate redirect policy during later integration.

Request cancellation and deadlines use the caller context. A per-request context timeout comes from `agent.request_timeout`, which defaults to `10s` and must remain shorter than `agent.heartbeat_interval`.

The initial successful status contract is deliberately limited to:

```text
200 OK
202 Accepted
204 No Content
```

Other statuses return a typed rejection error. Response-body reads are capped at 8 KiB, retained error messages are further bounded, and server-provided `Content-Length` does not control allocation. Server request IDs are read from `X-Request-ID`.

The runtime does not use this transport yet, so running the agent sends no heartbeat. There is no timer, scheduler, retry, backoff, registration, or authentication. Transport support is a foundation for the next integration step.

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
version: dev
commit: unknown
date: unknown

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
```

The `run` command loads validated configuration, creates the runtime skeleton, and waits for SIGINT or SIGTERM. The runtime performs no operational work yet. The `validate-config` command validates a file without starting the runtime, and `print-capabilities` reports only implemented CLI-level capabilities.

## Current Limitations

The agent still does not include heartbeat transmission, server communication, registration, authentication, collectors, controlled actions, or production installation. Linux collection, systemd monitoring, process monitoring, log shipping, Docker, and Kubernetes support also remain unimplemented.

## Roadmap

1. Repository and CLI foundation.
2. Configuration, structured logging, versioning, and graceful shutdown.
3. Persistent identity and heartbeat communication.
4. Linux host collection.
5. Additional controlled collectors and secure actions.

## License

Licensed under the Apache License 2.0. See LICENSE.
