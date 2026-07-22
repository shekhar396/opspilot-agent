# OpsPilot Agent

> The project is in early development and is not production-ready.

## Overview

OpsPilot Agent is planned to run on Linux virtual machines and eventually communicate with OpsPilot AI. Communication, evidence collection, registration, authentication, and action execution are not implemented in Step 2.

> OpsPilot Agent is a lightweight Linux operations agent. It collects approved operational evidence and communicates with OpsPilot AI. AI reasoning does not run inside the agent.

## What OpsPilot Agent Is

OpsPilot Agent is intended to become:

- A lightweight Linux host agent.
- A collector of explicitly approved operational evidence.
- A securely communicating component of the OpsPilot ecosystem.
- A controlled executor of predefined allow-listed actions in later milestones.

These capabilities are planned and are not implemented in Step 2.

## What OpsPilot Agent Is Not

OpsPilot Agent is not:

- An AI model.
- An autonomous AI agent.
- A general-purpose remote shell.
- A replacement for Prometheus.
- A full log-management platform.
- A Kubernetes agent at this stage.

## Current Scope

Step 2 currently provides:

- Initial Go module and public repository structure.
- Cobra-based CLI foundation.
- Explicit command constructors.
- Placeholder `run` command.
- Build-injectable version metadata.
- Configuration validation placeholder.
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

HTTPS communication, persistent identity, heartbeat, collectors, and controlled actions are future milestones and are not implemented in Step 2.

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

$ opspilot-agent validate-config
Configuration validation is not implemented yet

$ opspilot-agent print-capabilities
cli
version
```

The `run` command does not start an agent runtime yet. The `validate-config` command does not read or validate configuration yet. The `print-capabilities` command reports only implemented CLI-level capabilities.

## Current Limitations

Identity, heartbeat, collectors, networking, and controlled actions remain unimplemented. Step 2 also does not include configuration loading, Linux host inspection, authentication, production installation, Docker support, or Kubernetes support.

## Roadmap

1. Repository and CLI foundation.
2. Configuration, structured logging, versioning, and graceful shutdown.
3. Persistent identity and heartbeat communication.
4. Linux host collection.
5. Additional controlled collectors and secure actions.

## License

Licensed under the Apache License 2.0. See LICENSE.
