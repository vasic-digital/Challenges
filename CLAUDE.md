# CLAUDE.md - Challenges Module

## Overview

`digital.vasic.challenges` is a generic, reusable Go module for defining, registering, executing, and reporting on challenges (structured test scenarios). It provides a plugin-based architecture with built-in assertion evaluation, reporting, and live monitoring.

**Module**: `digital.vasic.challenges` (Go 1.24+)
**Depends on**: `digital.vasic.containers`

## Build & Test

```bash
go build ./...
go test ./... -count=1 -race
go test ./... -short              # Unit tests only
go test -tags=integration ./...   # Integration tests
go test -bench=. ./tests/benchmark/
```

## Code Style

- Standard Go conventions, `gofmt` formatting
- Imports grouped: stdlib, third-party, internal (blank line separated)
- Line length ≤ 100 chars
- Naming: `camelCase` private, `PascalCase` exported, acronyms all-caps
- Errors: always check, wrap with `fmt.Errorf("...: %w", err)`
- Tests: table-driven, `testify`, naming `Test<Struct>_<Method>_<Scenario>`

## Package Structure

| Package | Purpose |
|---------|---------|
| `pkg/challenge` | Core types: Challenge interface, Config, Result, BaseChallenge |
| `pkg/registry` | Challenge registration, dependency ordering (Kahn's algo) |
| `pkg/runner` | Execution engine (sequential, parallel, pipeline) |
| `pkg/assertion` | Assertion engine with 16 built-in evaluators |
| `pkg/report` | Report generation (Markdown, JSON, HTML) |
| `pkg/logging` | Structured logging (JSON, Console, Multi, Redacting) |
| `pkg/env` | Environment variable handling with redaction |
| `pkg/bank` | Challenge bank (load definitions from JSON/YAML) |
| `pkg/monitor` | Live monitoring with WebSocket dashboard |
| `pkg/metrics` | Prometheus-compatible challenge metrics |
| `pkg/plugin` | Plugin system for custom challenge types |
| `pkg/infra` | Infrastructure bridge to Containers module |

## Key Interfaces

- `challenge.Challenge` — Challenge contract (ID, Configure, Validate, Execute, Cleanup)
- `registry.Registry` — Challenge registration and dependency ordering
- `runner.Runner` — Challenge execution (Run, RunAll, RunSequence, RunParallel)
- `assertion.Engine` — Assertion evaluation with custom evaluators
- `report.Reporter` — Report generation (Markdown, JSON, HTML)
- `logging.Logger` — Structured logging with API request/response tracking
- `env.Loader` — Environment variable management with redaction
- `plugin.Plugin` — Plugin interface for extending the framework
- `infra.InfraProvider` — Bridge to container infrastructure

## Design Patterns

- **Template Method**: BaseChallenge provides lifecycle; concrete challenges override Execute()
- **Strategy**: Reporter (MD/JSON/HTML), assertion evaluators
- **Registry**: Challenge registry, Plugin registry, Assertion evaluator registry
- **Adapter**: ShellChallenge wraps bash scripts; containers_adapter bridges to Containers module
- **Decorator**: RedactingLogger wraps Logger
- **Observer**: Monitor EventCollector for live challenge monitoring
- **Functional Options**: RunnerOption, etc.

## Built-in Assertion Evaluators

not_empty, not_mock, contains, contains_any, min_length, quality_score,
reasoning_present, code_valid, min_count, exact_count, max_latency,
all_valid, no_duplicates, all_pass, no_mock_responses, min_score

## Commit Style

Conventional Commits: `feat(assertion): add custom evaluator support`
