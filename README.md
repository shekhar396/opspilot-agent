# OpsPilot Agent

> The project is in early development and is not production-ready.

## Overview

OpsPilot Agent is planned to run on Linux virtual machines and eventually communicate with OpsPilot AI. Communication, evidence collection, registration, authentication, and action execution are not implemented in Step 3.

> OpsPilot Agent is a lightweight Linux operations agent. It collects approved operational evidence and communicates with OpsPilot AI. AI reasoning does not run inside the agent.

## What OpsPilot Agent Is

OpsPilot Agent is intended to become:

- A lightweight Linux host agent.
- A collector of explicitly approved operational evidence.
- A securely communicating component of the OpsPilot ecosystem.
- A controlled executor of predefined allow-listed actions in later milestones.

These capabilities are planned and are not implemented in Step 3.

## What OpsPilot Agent Is Not

OpsPilot Agent is not:

- An AI model.
- An autonomous AI agent.
- A general-purpose remote shell.
- A replacement for Prometheus.
- A full log-management platform.
- A Kubernetes agent at this stage.

## Current Scope

Step 3 currently provides:

- Initial Go module and public repository structure.
- Cobra-based CLI foundation.
- Explicit command constructors.
- Placeholder `run` command.
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

HTTPS communication, persistent identity, heartbeat, collectors, and controlled actions are future milestones and are not implemented in Step 3.

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

logging:
  level: info
  format: json
```

Unknown fields and multiple YAML documents are rejected. `agent.name` accepts only letters, numbers, periods, underscores, and hyphens, with a maximum length of 128 characters. `agent.server_url` must be an HTTPS URL without credentials, query parameters, fragments, or a non-root path. `agent.heartbeat_interval` must be between `5s` and `1h`.

Supported logging levels are `debug`, `info`, `warn`, and `error`. Supported logging formats are `json` and `text`. These values are case-sensitive. Structured runtime logging is not implemented yet, and the current schema does not support secrets.

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

Configuration is validated but is not yet used by a running agent.

## CLI Usage

```bash
go run ./cmd/opspilot-agent --help
go run ./cmd/opspilot-agent run
go run ./cmd/opspilot-agent version
go run ./cmd/opspilot-agent validate-config
go run ./cmd/opspilot-agent print-capabilities
```

Current command output:

```text
$ opspilot-agent run
OpsPilot Agent runtime is not implemented yet

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
```

The `run` command remains a placeholder and does not start an agent runtime. The `validate-config` command validates a file without starting the agent. The `print-capabilities` command reports only implemented CLI-level capabilities.

## Current Limitations

Identity, heartbeat, collectors, server communication, networking, and controlled actions remain unimplemented. Step 3 also does not include structured logging, Linux host inspection, authentication, production installation, Docker support, or Kubernetes support.

## Roadmap

1. Repository and CLI foundation.
2. Configuration, structured logging, versioning, and graceful shutdown.
3. Persistent identity and heartbeat communication.
4. Linux host collection.
5. Additional controlled collectors and secure actions.

## License

Licensed under the Apache License 2.0. See LICENSE.
