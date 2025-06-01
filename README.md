# CI-Driven Background Agent Orchestrator (Amp Companion)

A lightweight orchestrator API and CLI for managing CI-driven Amp tasks from a single terminal.

## Overview

This project provides:
- **Orchestrator API** (Go + Gin): REST API for task management
- **CLI Tool** (`ampx`): Command-line interface for developers
- **Worker**: Background processor for CI-driven tasks

## Quick Start

### Prerequisites
- Go 1.22+
- Docker (optional, for containerized deployment)
- GitHub CLI (`gh`) for CI integration
- Amp CLI for code generation

### Installation

```bash
# Clone the repository
git clone https://github.com/brettsmith212/ci-test-2.git
cd ci-test-2

# Install dependencies
go mod tidy

# Build binaries
go build -o bin/orchestrator ./cmd/orchestrator
go build -o bin/ampx ./cmd/ampx
go build -o bin/worker ./cmd/worker
```

### Usage

#### Start the Orchestrator API
```bash
./bin/orchestrator
```

#### Use the CLI
```bash
# Start a new task
./bin/ampx start --repo git@github.com:acme/api.git --task "Migrate tests to Vitest"

# List tasks
./bin/ampx ls

# View task logs
./bin/ampx logs <task-id>

# Continue a failed task
./bin/ampx continue <task-id> -m "Fix test configuration"

# Abort a running task
./bin/ampx abort <task-id>
```

## Architecture

- **API Server**: Manages task lifecycle and persistence
- **CLI Client**: User interface for task operations
- **Worker**: Executes tasks with Amp integration and CI monitoring
- **SQLite Database**: Stores task metadata and status

## Development

```bash
# Run tests
go test ./...

# Run API server in development
go run cmd/orchestrator/main.go

# Run CLI commands
go run cmd/ampx/main.go <command>
```

## License

MIT License - see LICENSE file for details.
