# Contributing to Challenges

## Prerequisites

- Go 1.24 or later
- Git with SSH access configured
- Docker/Podman (for integration tests)

## Getting Started

1. Clone: `git clone <ssh-url> && cd Challenges`
2. Build: `go build ./...`
3. Test: `go test ./... -count=1 -race`

## Commit Conventions

```
<type>(<scope>): <description>

feat(assertion): add regex evaluator
fix(runner): prevent timeout race
test(registry): add cycle detection tests
```

Scopes: `challenge`, `registry`, `runner`, `assertion`, `report`, `plugin`, `monitor`

## Adding New Assertion Evaluator

1. Add to `pkg/assertion/evaluators.go`:
```go
func RegexMatchEvaluator(value, expected interface{}) (bool, error) {
    pattern := expected.(string)
    text := value.(string)
    matched, err := regexp.MatchString(pattern, text)
    return matched, err
}
```

2. Register in `NewEngine()`:
```go
engine.RegisterEvaluator("regex_match", RegexMatchEvaluator)
```

3. Test in `pkg/assertion/evaluators_test.go`

## Testing

```bash
go test ./... -count=1 -race      # All tests
go test ./... -short               # Unit tests only
go test -tags=integration ./...    # Integration tests
go test -bench=. ./tests/benchmark/
```

## Pull Request Process

1. Branch: `feat/your-feature`
2. Commit with conventional commits
3. Pass all tests: `go test ./... -count=1 -race`
4. Format: `gofmt -w .`
5. Create PR against `main`

---

**Last Updated**: February 10, 2026
