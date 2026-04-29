# AGENTS.md - Challenges Module

## MANDATORY: Project-Agnostic / 100% Decoupled

**This module MUST remain 100% decoupled from any consuming project. It is designed for generic use with ANY project, not one specific consumer.**

- NEVER hardcode project-specific package names, endpoints, device serials, or region-specific data
- NEVER import anything from a consuming project
- NEVER add project-specific defaults, presets, or fixtures into source code
- All project-specific data MUST be registered by the caller via public APIs — never baked into the library
- Default values MUST be empty or generic

Violations void the release. Refactor to restore generic behaviour before any commit.

## MANDATORY: No CI/CD Pipelines

**NO GitHub Actions, GitLab CI/CD, or any automated pipeline may exist in this repository!**

- No `.github/workflows/` directory
- No `.gitlab-ci.yml` file
- No Jenkinsfile, .travis.yml, .circleci, or any other CI configuration
- All builds and tests are run manually or via Makefile targets
- This rule is permanent and non-negotiable

## Module Overview

`digital.vasic.challenges` is a generic, reusable Go module for defining, registering, executing, and reporting on challenges (structured test scenarios). It provides a comprehensive framework for validation testing with built-in assertion evaluation, multiple reporting formats, live monitoring, and plugin extensibility.

**Module path**: `digital.vasic.challenges`
**Go version**: 1.24+
**Dependencies**: `digital.vasic.containers` (infrastructure bridge), standard Go libraries, testify (tests only)

## Package Responsibilities

| Package | Path | Responsibility |
|---------|------|----------------|
| `challenge` | `pkg/challenge/` | Core types and interfaces: `Challenge` interface defining lifecycle (Configure, Validate, Execute, Cleanup). `BaseChallenge` template implementation. `Config` and `Result` structures. Challenge status enumeration. |
| `registry` | `pkg/registry/` | Challenge registration and dependency management: Registry with topological sorting (Kahn's algorithm). Dependency validation (cycle detection). Challenge lookup by ID/tags. Ordered execution sequencing. |
| `runner` | `pkg/runner/` | Execution engine: Sequential, parallel, and pipeline execution modes. Concurrent execution with semaphore control. Timeout handling and graceful cancellation. Result aggregation and reporting. |
| `assertion` | `pkg/assertion/` | Assertion evaluation engine: 16 built-in evaluators (`not_empty`, `contains`, `min_length`, `quality_score`, etc.). Custom evaluator registration. Expression parser for complex assertions. Flexible comparison operators. |
| `report` | `pkg/report/` | Report generation: Multiple formats (Markdown, JSON, HTML). Summary statistics (pass/fail/skip counts). Detailed execution timelines. Export to file or string. |
| `logging` | `pkg/logging/` | Structured logging: JSON and console formatters. Multi-logger composition. Redacting logger for sensitive data. API request/response tracking. |
| `env` | `pkg/env/` | Environment variable management: Load from .env files. Variable interpolation and defaults. Redaction patterns for secrets (API keys, tokens). Validation and type conversion. |
| `bank` | `pkg/bank/` | Challenge bank (definition loading): Load challenge definitions from JSON/YAML. Template variable substitution. Bulk challenge instantiation. Definition validation. |
| `monitor` | `pkg/monitor/` | Live monitoring: WebSocket-based real-time dashboard. Event collection (challenge start/complete/fail). Progress tracking and ETA calculation. Metrics export. |
| `metrics` | `pkg/metrics/` | Prometheus-compatible metrics: Challenge execution counters. Duration histograms. Success/failure rates. Custom metric registration. |
| `plugin` | `pkg/plugin/` | Plugin system: Plugin interface for custom challenge types. Dynamic plugin loading. Lifecycle management (Init, Shutdown). Plugin registry and versioning. |
| `infra` | `pkg/infra/` | Infrastructure bridge: Adapter to `digital.vasic.containers` module. Service startup/shutdown coordination. Health check integration. Resource cleanup. |

## Dependency Graph

```
runner  --->  challenge, registry, assertion, report, logging
registry  --->  challenge
assertion  --->  logging
report  --->  challenge, logging
monitor  --->  challenge, metrics
plugin  --->  challenge, registry
infra  --->  challenge, containers module
bank  --->  challenge, env
```

`challenge` is the foundational package. `runner` integrates most packages for orchestration.

## Key Files

| File | Purpose |
|------|---------|
| `pkg/challenge/challenge.go` | Challenge interface, BaseChallenge, Config, Result types |
| `pkg/challenge/shell.go` | ShellChallenge for executing bash scripts |
| `pkg/registry/registry.go` | Registry implementation with dependency ordering |
| `pkg/runner/runner.go` | Runner implementation with execution modes |
| `pkg/assertion/engine.go` | Assertion engine with built-in evaluators |
| `pkg/assertion/evaluators.go` | All 16 built-in evaluator implementations |
| `pkg/report/markdown.go` | Markdown report generator |
| `pkg/report/json.go` | JSON report generator |
| `pkg/report/html.go` | HTML report generator |
| `pkg/logging/logger.go` | Logger interface and implementations |
| `pkg/env/loader.go` | Environment variable loader with redaction |
| `pkg/bank/bank.go` | Challenge bank with JSON/YAML loading |
| `pkg/monitor/monitor.go` | Live monitoring with WebSocket server |
| `pkg/metrics/metrics.go` | Prometheus metrics collector |
| `pkg/plugin/plugin.go` | Plugin interface and registry |
| `pkg/infra/adapter.go` | Containers module adapter |
| `go.mod` | Module definition and dependencies |
| `CLAUDE.md` | AI coding assistant instructions |
| `README.md` | User-facing documentation with quick start |

## Agent Coordination Guide

### Division of Work

When multiple agents work on this module simultaneously, divide work by package boundary:

1. **Challenge Agent** -- Owns `pkg/challenge/`. Core types affect all other packages. Must coordinate before modifying `Challenge` interface or `Result` structure.
2. **Registry Agent** -- Owns `pkg/registry/`. Dependency ordering logic. Changes rarely affect other packages except runner.
3. **Runner Agent** -- Owns `pkg/runner/`. Integration layer. Requires testing against all execution modes.
4. **Assertion Agent** -- Owns `pkg/assertion/`. New evaluators can be added independently. Evaluator registry changes require runner updates.
5. **Report Agent** -- Owns `pkg/report/`. New report formats can be added independently. Reporter interface changes affect runner.
6. **Monitoring Agent** -- Owns `pkg/monitor/`. Real-time monitoring. Can work independently but coordinates with runner for event hooks.
7. **Plugin Agent** -- Owns `pkg/plugin/`. Plugin system. Must coordinate with registry for plugin registration.

### Coordination Rules

- **Challenge interface changes** require all agents to update. The `Challenge` interface is the shared contract.
- **Assertion evaluators** and **report formats** are independent and can be modified in parallel.
- **Runner package** integrates all packages. Any interface change in sub-packages requires corresponding runner updates.
- **Monitor and metrics** packages are loosely coupled. Coordinate on event schema.
- **Test isolation**: Each package has its own `_test.go` files. Integration tests in `runner` package.
- **No circular dependencies**: The dependency graph is strictly acyclic. Never import `runner` from sub-packages.

### Safe Parallel Changes

These changes can be made simultaneously without coordination:
- Adding a new assertion evaluator to `pkg/assertion/`
- Adding a new report format to `pkg/report/`
- Adding new monitoring events to `pkg/monitor/`
- Adding new plugins to `pkg/plugin/`
- Adding new tests to any package
- Updating documentation

### Changes Requiring Coordination

- Modifying the `Challenge` interface methods
- Changing `Result` structure fields
- Modifying assertion evaluator registry interface
- Adding new execution modes to runner
- Changing event schema in monitor
- Modifying plugin interface

## Build and Test Commands

```bash
# Build all packages
go build ./...

# Run all tests with race detection
go test ./... -count=1 -race

# Run unit tests only (short mode)
go test ./... -short

# Run integration tests (requires Containers module)
go test -tags=integration ./...

# Run benchmarks
go test -bench=. ./tests/benchmark/

# Run a specific test
go test -v -run TestRunner_RunAll ./pkg/runner/

# Format code
gofmt -w .

# Vet code
go vet ./...
```

## Commit Conventions

Follow Conventional Commits with package scope:

```
feat(assertion): add regex_match evaluator
feat(report): add PDF report generator
feat(plugin): implement plugin versioning
fix(runner): prevent race condition in parallel execution
fix(registry): detect circular dependencies correctly
test(assertion): add evaluator edge case tests
docs(challenges): update plugin development guide
refactor(monitor): extract WebSocket server to separate file
```

## Thread Safety Notes

- **Runner** executes challenges concurrently with semaphore control. Uses `sync.WaitGroup` for coordination.
- **Registry** is thread-safe for reads after initialization. Writes during registration use mutex.
- **Assertion engine** evaluators must be safe for concurrent invocation.
- **Monitor** uses channels for event collection and mutex for state access.
- **Metrics collector** uses atomic operations for counters.
- **Plugin registry** locks during plugin loading/unloading.

## Built-in Assertion Evaluators

| Evaluator | Description | Example |
|-----------|-------------|---------|
| `not_empty` | Value must not be empty string or nil | `"result": { "not_empty": true }` |
| `not_mock` | Response must not be a mock/placeholder | `"response": { "not_mock": true }` |
| `contains` | String contains substring | `"output": { "contains": "success" }` |
| `contains_any` | String contains any of the substrings | `"output": { "contains_any": ["ok", "success"] }` |
| `min_length` | String has minimum length | `"response": { "min_length": 10 }` |
| `quality_score` | LLM response quality score (0-1) | `"quality": { "quality_score": 0.8 }` |
| `reasoning_present` | Response contains reasoning/explanation | `"answer": { "reasoning_present": true }` |
| `code_valid` | Code block is syntactically valid | `"code": { "code_valid": "go" }` |
| `min_count` | Array/collection minimum count | `"items": { "min_count": 5 }` |
| `exact_count` | Array/collection exact count | `"items": { "exact_count": 10 }` |
| `max_latency` | Operation completed within time limit (ms) | `"latency": { "max_latency": 1000 }` |
| `all_valid` | All items in array pass validation | `"results": { "all_valid": true }` |
| `no_duplicates` | Array has no duplicate values | `"ids": { "no_duplicates": true }` |
| `all_pass` | All nested assertions pass | `"tests": { "all_pass": true }` |
| `no_mock_responses` | No mock/placeholder responses in collection | `"responses": { "no_mock_responses": true }` |
| `min_score` | Numeric score meets minimum threshold | `"score": { "min_score": 85.0 }` |

## Configuration Example

```go
package main

import (
    "context"
    "digital.vasic.challenges/pkg/challenge"
    "digital.vasic.challenges/pkg/registry"
    "digital.vasic.challenges/pkg/runner"
    "digital.vasic.challenges/pkg/report"
)

func main() {
    // Create registry
    reg := registry.New()

    // Register challenges
    reg.Register(&MyChallenge{
        id: "test-api",
        dependencies: []string{},
    })

    // Create runner
    run := runner.New(reg,
        runner.WithParallelism(5),
        runner.WithTimeout(30*time.Second),
    )

    // Execute all challenges
    results, _ := run.RunAll(context.Background())

    // Generate report
    reporter := report.NewMarkdownReporter()
    report := reporter.Generate(results)
    fmt.Println(report)
}
```

## Custom Challenge Example

```go
type APITestChallenge struct {
    challenge.BaseChallenge
    endpoint string
}

func (c *APITestChallenge) Execute(ctx context.Context) error {
    resp, err := http.Get(c.endpoint)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    // Evaluate assertions
    return c.AssertionEngine.Evaluate(ctx, map[string]interface{}{
        "status_code": resp.StatusCode,
        "response": map[string]interface{}{
            "not_empty": true,
            "contains": "success",
        },
    })
}
```

## Best Practices

### 1. Use Dependency Ordering
```go
// Good - declare dependencies
challenge := &MyChallenge{
    dependencies: []string{"setup-database", "start-api"},
}

// Registry will execute in correct order
```

### 2. Implement Cleanup
```go
func (c *MyChallenge) Cleanup(ctx context.Context) error {
    // Always clean up resources
    c.client.Close()
    c.db.Close()
    return nil
}
```

### 3. Use Assertion Engine
```go
// Good - use built-in evaluators
return c.AssertionEngine.Evaluate(ctx, map[string]interface{}{
    "result": {
        "not_empty": true,
        "min_length": 10,
    },
})

// Bad - manual validation
if result == "" || len(result) < 10 {
    return errors.New("validation failed")
}
```

### 4. Set Timeouts
```go
// Good - always set timeouts
runner := runner.New(reg,
    runner.WithTimeout(30*time.Second),
)

// Bad - no timeout (can hang indefinitely)
runner := runner.New(reg)
```

### 5. Use Structured Logging
```go
// Good - structured logging
logger.InfoWithFields("Challenge executed", map[string]interface{}{
    "challenge_id": c.ID(),
    "duration": elapsed,
    "status": "passed",
})

// Bad - unstructured logging
log.Printf("Challenge %s took %v and passed", c.ID(), elapsed)
```

---

**Last Updated**: February 10, 2026
**Version**: 1.0.0
**Status**: ✅ Production Ready

### ⚠️⚠️⚠️ ABSOLUTELY MANDATORY: ZERO UNFINISHED WORK POLICY

NO unfinished work, TODOs, or known issues may remain in the codebase. EVER.

PROHIBITED: TODO/FIXME comments, empty implementations, silent errors, fake data, unwrap() calls that panic, empty catch blocks.

REQUIRED: Fix ALL issues immediately, complete implementations before committing, proper error handling in ALL code paths, real test assertions.

Quality Principle: If it is not finished, it does not ship. If it ships, it is finished.

## ⚠️ MANDATORY: NO SUDO OR ROOT EXECUTION

**ALL operations MUST run at local user level ONLY.**

This is a PERMANENT and NON-NEGOTIABLE security constraint:

- **NEVER** use `sudo` in ANY command
- **NEVER** use `su` in ANY command
- **NEVER** execute operations as `root` user
- **NEVER** elevate privileges for file operations
- **ALL** infrastructure commands MUST use user-level container runtimes (rootless podman/docker)
- **ALL** file operations MUST be within user-accessible directories
- **ALL** service management MUST be done via user systemd or local process management
- **ALL** builds, tests, and deployments MUST run as the current user

### Container-Based Solutions
When a build or runtime environment requires system-level dependencies, use containers instead of elevation:

- **Use the `Containers` submodule** (`https://github.com/vasic-digital/Containers`) for containerized build and runtime environments
- **Add the `Containers` submodule as a Git dependency** and configure it for local use within the project
- **Build and run inside containers** to avoid any need for privilege escalation
- **Rootless Podman/Docker** is the preferred container runtime

### Why This Matters
- **Security**: Prevents accidental system-wide damage
- **Reproducibility**: User-level operations are portable across systems
- **Safety**: Limits blast radius of any issues
- **Best Practice**: Modern container workflows are rootless by design

### When You See SUDO
If any script or command suggests using `sudo` or `su`:
1. STOP immediately
2. Find a user-level alternative
3. Use rootless container runtimes
4. Use the `Containers` submodule for containerized builds
5. Modify commands to work within user permissions

**VIOLATION OF THIS CONSTRAINT IS STRICTLY PROHIBITED.**

<!-- BEGIN host-power-management addendum (CONST-033) -->

## Host Power Management — Hard Ban (CONST-033)

**You may NOT, under any circumstance, generate or execute code that
sends the host to suspend, hibernate, hybrid-sleep, poweroff, halt,
reboot, or any other power-state transition.** This rule applies to:

- Every shell command you run via the Bash tool.
- Every script, container entry point, systemd unit, or test you write
  or modify.
- Every CLI suggestion, snippet, or example you emit.

**Forbidden invocations** (non-exhaustive — see CONST-033 in
`CONSTITUTION.md` for the full list):

- `systemctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot|kexec`
- `loginctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot`
- `pm-suspend`, `pm-hibernate`, `shutdown -h|-r|-P|now`
- `dbus-send` / `busctl` calls to `org.freedesktop.login1.Manager.Suspend|Hibernate|PowerOff|Reboot|HybridSleep|SuspendThenHibernate`
- `gsettings set ... sleep-inactive-{ac,battery}-type` to anything but `'nothing'` or `'blank'`

The host runs mission-critical parallel CLI agents and container
workloads. Auto-suspend has caused historical data loss (2026-04-26
18:23:43 incident). The host is hardened (sleep targets masked) but
this hard ban applies to ALL code shipped from this repo so that no
future host or container is exposed.

**Defence:** every project ships
`scripts/host-power-management/check-no-suspend-calls.sh` (static
scanner) and
`challenges/scripts/no_suspend_calls_challenge.sh` (challenge wrapper).
Both MUST be wired into the project's CI / `run_all_challenges.sh`.

**Full background:** `docs/HOST_POWER_MANAGEMENT.md` and `CONSTITUTION.md` (CONST-033).

<!-- END host-power-management addendum (CONST-033) -->



<!-- CONST-035 anti-bluff addendum (cascaded) -->

## CONST-035 — Anti-Bluff Tests & Challenges (mandatory; inherits from root)

Tests and Challenges in this submodule MUST verify the product, not
the LLM's mental model of the product. A test that passes when the
feature is broken is worse than a missing test — it gives false
confidence and lets defects ship to users. Functional probes at the
protocol layer are mandatory:

- TCP-open is the FLOOR, not the ceiling. Postgres → execute
  `SELECT 1`. Redis → `PING` returns `PONG`. ChromaDB → `GET
  /api/v1/heartbeat` returns 200. MCP server → TCP connect + valid
  JSON-RPC handshake. HTTP gateway → real request, real response,
  non-empty body.
- Container `Up` is NOT application healthy. A `docker/podman ps`
  `Up` status only means PID 1 is running; the application may be
  crash-looping internally.
- No mocks/fakes outside unit tests (already CONST-030; CONST-035
  raises the cost of a mock-driven false pass to the same severity
  as a regression).
- Re-verify after every change. Don't assume a previously-passing
  test still verifies the same scope after a refactor.
- Verification of CONST-035 itself: deliberately break the feature
  (e.g. `kill <service>`, swap a password). The test MUST fail. If
  it still passes, the test is non-conformant and MUST be tightened.

## CONST-033 clarification — distinguishing host events from sluggishness

Heavy container builds (BuildKit pulling many GB of layers, parallel
podman/docker compose-up across many services) can make the host
**appear** unresponsive — high load average, slow SSH, watchers
timing out. **This is NOT a CONST-033 violation.** Suspend / hibernate
/ logout are categorically different events. Distinguish via:

- `uptime` — recent boot? if so, the host actually rebooted.
- `loginctl list-sessions` — session(s) still active? if yes, no logout.
- `journalctl ... | grep -i 'will suspend\|hibernate'` — zero broadcasts
  since the CONST-033 fix means no suspend ever happened.
- `dmesg | grep -i 'killed process\|out of memory'` — OOM kills are
  also NOT host-power events; they're memory-pressure-induced and
  require their own separate fix (lower per-container memory limits,
  reduce parallelism).

A sluggish host under build pressure recovers when the build finishes;
a suspended host requires explicit unsuspend (and CONST-033 should
make that impossible by hardening `IdleAction=ignore` +
`HandleSuspendKey=ignore` + masked `sleep.target`,
`suspend.target`, `hibernate.target`, `hybrid-sleep.target`).

If you observe what looks like a suspend during heavy builds, the
correct first action is **not** "edit CONST-033" but `bash
challenges/scripts/host_no_auto_suspend_challenge.sh` to confirm the
hardening is intact. If hardening is intact AND no suspend
broadcast appears in journal, the perceived event was build-pressure
sluggishness, not a power transition.

<!-- BEGIN no-session-termination addendum (CONST-036) -->

## User-Session Termination — Hard Ban (CONST-036)

**You may NOT, under any circumstance, generate or execute code that
ends the currently-logged-in user's desktop session, kills their
`user@<UID>.service` user manager, or indirectly forces them to
manually log out / power off.** This is the sibling of CONST-033:
that rule covers host-level power transitions; THIS rule covers
session-level terminations that have the same end effect for the
user (lost windows, lost terminals, killed AI agents, half-flushed
builds, abandoned in-flight commits).

**Why this rule exists.** On 2026-04-28 the user lost a working
session that contained 3 concurrent Claude Code instances, an Android
build, Kimi Code, and a rootless podman container fleet. The
`user.slice` consumed 60.6 GiB peak / 5.2 GiB swap, the GUI became
unresponsive, the user was forced to log out and then power off via
the GNOME shell. The host could not auto-suspend (CONST-033 was in
place and verified) and the kernel OOM killer never fired — but the
user had to manually end the session anyway, because nothing
prevented overlapping heavy workloads from saturating the slice.
CONST-036 closes that loophole at both the source-code layer and the
operational layer. See
`docs/issues/fixed/SESSION_LOSS_2026-04-28.md` in the HelixAgent
project.

**Forbidden direct invocations** (non-exhaustive):

- `loginctl terminate-user|terminate-session|kill-user|kill-session`
- `systemctl stop user@<UID>` / `systemctl kill user@<UID>`
- `gnome-session-quit`
- `pkill -KILL -u $USER` / `killall -u $USER`
- `dbus-send` / `busctl` calls to `org.gnome.SessionManager.Logout|Shutdown|Reboot`
- `echo X > /sys/power/state`
- `/usr/bin/poweroff`, `/usr/bin/reboot`, `/usr/bin/halt`

**Indirect-pressure clauses:**

1. Do not spawn parallel heavy workloads casually; check `free -h`
   first; keep `user.slice` under 70% of physical RAM.
2. Long-lived background subagents go in `system.slice`. Rootless
   podman containers die with the user manager.
3. Document AI-agent concurrency caps in CLAUDE.md.
4. Never script "log out and back in" recovery flows.

**Defence:** every project ships
`scripts/host-power-management/check-no-session-termination-calls.sh`
(static scanner) and
`challenges/scripts/no_session_termination_calls_challenge.sh`
(challenge wrapper). Both MUST be wired into the project's CI /
`run_all_challenges.sh`.

<!-- END no-session-termination addendum (CONST-036) -->

<!-- BEGIN const035-strengthening-2026-04-29 -->

## CONST-035 — End-User Usability Mandate (2026-04-29 strengthening)

A test or Challenge that PASSES is a CLAIM that the tested behavior
**works for the end user of the product**. The HelixAgent project
has repeatedly hit the failure mode where every test ran green AND
every Challenge reported PASS, yet most product features did not
actually work — buggy challenge wrappers masked failed assertions,
scripts checked file existence without executing the file,
"reachability" tests tolerated timeouts, contracts were honest in
advertising but broken in dispatch. **This MUST NOT recur.**

Every PASS result MUST guarantee:

a. **Quality** — the feature behaves correctly under inputs an end
   user will send, including malformed input, edge cases, and
   concurrency that real workloads produce.
b. **Completion** — the feature is wired end-to-end from public
   API surface down to backing infrastructure, with no stub /
   placeholder / "wired lazily later" gaps that silently 503.
c. **Full usability** — a CLI agent / SDK consumer / direct curl
   client following the documented model IDs, request shapes, and
   endpoints SUCCEEDS without having to know which of N internal
   aliases the dispatcher actually accepts.

A passing test that doesn't certify all three is a **bluff** and
MUST be tightened, or marked `t.Skip("...SKIP-OK: #<ticket>")`
so absence of coverage is loud rather than silent.

### Bluff taxonomy (each pattern observed in HelixAgent and now forbidden)

- **Wrapper bluff** — assertions PASS but the wrapper's exit-code
  logic is buggy, marking the run FAILED (or the inverse: assertions
  FAIL but the wrapper swallows them). Every aggregating wrapper MUST
  use a robust counter (`! grep -qs "|FAILED|" "$LOG"` style) —
  never inline arithmetic on a command that prints AND exits
  non-zero.
- **Contract bluff** — the system advertises a capability but
  rejects it in dispatch. Every advertised capability MUST be
  exercised by a test or Challenge that actually invokes it.
- **Structural bluff** — `check_file_exists "foo_test.go"` passes
  if the file is present but doesn't run the test or assert anything
  about its content. File-existence checks MUST be paired with at
  least one functional assertion.
- **Comment bluff** — a code comment promises a behavior the code
  doesn't actually have. Documentation written before / about code
  MUST be re-verified against the code on every change touching the
  documented function.
- **Skip bluff** — `t.Skip("not running yet")` without a
  `SKIP-OK: #<ticket>` marker silently passes. Every skip needs the
  marker; CI fails on bare skips.

The taxonomy is illustrative, not exhaustive. Every Challenge or
test added going forward MUST pass an honest self-review against
this taxonomy before being committed.

<!-- END const035-strengthening-2026-04-29 -->

## MANDATORY ANTI-BLUFF COVENANT — END-USER QUALITY GUARANTEE (User mandate, 2026-04-28)

**Forensic anchor — direct user mandate (verbatim):**

> "We had been in position that all tests do execute with success and all Challenges as well, but in reality the most of the features does not work and can't be used! This MUST NOT be the case and execution of tests and Challenges MUST guarantee the quality, the completion and full usability by end users of the product!"

This is the historical origin of the project's anti-bluff covenant.
Every test, every Challenge, every gate, every mutation pair exists
to make the failure mode (PASS on broken-for-end-user feature)
mechanically impossible.

**Operative rule:** the bar for shipping is **not** "tests pass"
but **"users can use the feature."** Every PASS in this codebase
MUST carry positive evidence captured during execution that the
feature works for the end user. Metadata-only PASS, configuration-
only PASS, "absence-of-error" PASS, and grep-based PASS without
runtime evidence are all critical defects regardless of how green
the summary line looks.

**Tests AND Challenges (HelixQA) are bound equally** — a Challenge
that scores PASS on a non-functional feature is the same class of
defect as a unit test that does. Both must produce positive end-
user evidence; both are subject to the §8.1 five-constraint rule
and §11 captured-evidence requirement.

**Canonical authority:** parent
[`docs/guides/ATMOSPHERE_CONSTITUTION.md`](../../docs/guides/ATMOSPHERE_CONSTITUTION.md)
§8.1 (positive-evidence-only validation) + §11 (bleeding-edge
ultra-perfection quality bar) + §11.3 (the "no bluff" CLAUDE.md /
AGENTS.md mandate) + **§11.4 (this end-user-quality-guarantee
