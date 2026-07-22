# OpsPilot Agent

> The project is in early development and is not production-ready.

## Overview

OpsPilot Agent is planned to run on Linux virtual machines and eventually communicate with OpsPilot AI. Communication, evidence collection, registration, authentication, and action execution are not implemented in Step 1.

> OpsPilot Agent is a lightweight Linux operations agent. It collects approved operational evidence and communicates with OpsPilot AI. AI reasoning does not run inside the agent.

## What OpsPilot Agent Is

OpsPilot Agent is intended to become:

- A lightweight Linux host agent.
- A collector of explicitly approved operational evidence.
- A securely communicating component of the OpsPilot ecosystem.
- A controlled executor of predefined allow-listed actions in later milestones.

These capabilities are planned and are not implemented in Step 1.

## What OpsPilot Agent Is Not

OpsPilot Agent is not:

- An AI model.
- An autonomous AI agent.
- A general-purpose remote shell.
- A replacement for Prometheus.
- A full log-management platform.
- A Kubernetes agent at this stage.

## Current Scope

Step 1 currently provides only:

- Initial Go module.
- Minimal executable.
- Basic public repository structure.
- License.
- README foundation.

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

HTTPS communication, persistent identity, heartbeat, collectors, and controlled actions are future milestones and are not implemented in Step 1.

## Requirements

- Go installed locally
- Git
- Linux, macOS, or Windows for initial development
- Linux as the initial runtime target

## Build

```bash
go build -o bin/opspilot-agent ./cmd/opspilot-agent
```

## Run

```bash
./bin/opspilot-agent
```

Expected output:

```text
OpsPilot Agent development build
```

## Current Limitations

Step 1 does not yet include:

- Configuration.
- Agent identity.
- Heartbeats.
- Server communication.
- Linux collectors.
- systemd monitoring.
- Process monitoring.
- Health checks.
- Logs.
- Docker.
- Controlled actions.
- Authentication.
- Production installation.

## Roadmap

1. Repository and CLI foundation.
2. Configuration, structured logging, versioning, and graceful shutdown.
3. Persistent identity and heartbeat communication.
4. Linux host collection.
5. Additional controlled collectors and secure actions.

## License

Licensed under the Apache License 2.0. See LICENSE.
