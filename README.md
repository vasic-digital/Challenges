# digital.vasic.challenges

A generic, reusable Go module for defining, registering, executing, and reporting on challenges (structured test scenarios). Features a plugin-based architecture with built-in assertion evaluation, multi-format reporting, and live monitoring.

## Installation

```bash
go get digital.vasic.challenges
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "digital.vasic.challenges/pkg/assertion"
    "digital.vasic.challenges/pkg/challenge"
    "digital.vasic.challenges/pkg/registry"
    "digital.vasic.challenges/pkg/runner"
)

// Define a custom challenge
type HealthChallenge struct {
    challenge.BaseChallenge
}

func NewHealthChallenge() *HealthChallenge {
    return &HealthChallenge{
        BaseChallenge: *challenge.NewBaseChallenge(
            "health_check", "Health Check",
            "Verify all services are healthy", "core",
            nil,
        ),
    }
}

func (c *HealthChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
    result := c.CreateResult()
    result.Status = challenge.StatusPassed
    result.Assertions = []challenge.AssertionResult{
        {Type: "not_empty", Target: "response", Passed: true,
            Message: "Service responded"},
    }
    return result, nil
}

func main() {
    ctx := context.Background()

    // Register challenge
    reg := registry.NewRegistry()
    reg.Register(NewHealthChallenge())

    // Run all challenges
    r := runner.NewRunner(
        runner.WithRegistry(reg),
        runner.WithTimeout(5 * time.Minute),
    )

    results, err := r.RunAll(ctx, &challenge.Config{
        Verbose: true,
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, res := range results {
        fmt.Printf("%s: %s (%v)\n",
            res.ChallengeName, res.Status, res.Duration)
    }
}
```

## Features

- **Challenge framework**: Define, register, and execute structured test scenarios
- **Dependency ordering**: Automatic topological sort (Kahn's algorithm)
- **Assertion engine**: 16 built-in evaluators + custom evaluator support
- **Multi-format reports**: Markdown, JSON, HTML
- **Shell adapter**: Wrap existing bash scripts as challenges
- **Plugin system**: Extend with custom challenge types and assertions
- **Live monitoring**: WebSocket-based real-time dashboard
- **Prometheus metrics**: Built-in challenge metrics
- **Environment management**: Secure env var handling with redaction
- **Challenge banks**: Load definitions from JSON/YAML files
- **Parallel execution**: Run independent challenges concurrently
- **Infrastructure bridge**: Integrates with `digital.vasic.containers`

## Architecture

```
runner.Runner
├── registry.Registry            (Challenge registration + ordering)
├── assertion.Engine             (16 built-in evaluators)
├── report.Reporter              (Markdown/JSON/HTML)
├── logging.Logger               (Structured logging)
├── monitor.EventCollector       (Live monitoring)
└── plugin.PluginRegistry        (Extensibility)

challenge.Challenge (interface)
├── challenge.BaseChallenge      (Template method base)
├── challenge.ShellChallenge     (Bash script wrapper)
└── [your custom challenges]

infra.InfraProvider
└── ContainersAdapter            (Bridge to digital.vasic.containers)
```

## Built-in Assertion Evaluators

| Evaluator | Description |
|-----------|-------------|
| `not_empty` | Value is non-nil and non-empty |
| `not_mock` | Response is not mocked/placeholder |
| `contains` | String contains substring (case-insensitive) |
| `contains_any` | String contains any of the given values |
| `min_length` | String length meets minimum |
| `quality_score` | Numeric score meets threshold |
| `reasoning_present` | Response contains reasoning indicators |
| `code_valid` | Response contains valid code patterns |
| `min_count` | Count meets minimum |
| `exact_count` | Count matches exactly |
| `max_latency` | Response time within limit |
| `all_valid` | All array items are valid |
| `no_duplicates` | No duplicate items in array |
| `all_pass` | All sub-assertions pass |
| `no_mock_responses` | No mocked responses in array |
| `min_score` | Numeric minimum score |

## License

MIT
