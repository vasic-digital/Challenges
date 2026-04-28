# CLAUDE.md - Challenges Module


## Definition of Done

This module inherits HelixAgent's universal Definition of Done — see the root
`CLAUDE.md` and `docs/development/definition-of-done.md`. In one line: **no
task is done without pasted output from a real run of the real system in the
same session as the change.** Coverage and green suites are not evidence.

### Acceptance demo for this module

```bash
# Full challenge execution engine: Configure → Execute → Validate → Cleanup + report
cd Challenges && GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v \
  -run 'TestRunner_Execute' ./...
```
Expect: PASS; a sample challenge runs to completion, assertion engine evaluates, and JSON/Markdown reports are emitted. The `cmd/userflow-runner` CLI (see `Challenges/README.md`) drives the same pipeline end-to-end.


## MANDATORY HOST-SESSION SAFETY (Constitution §12)

**Forensic incident, 2026-04-27 22:22:14 (MSK):** the developer's
`user@1000.service` was SIGKILLed under an OOM cascade triggered by
`pip3 install --user openai-whisper` running on top of chronic
podman-pod memory pressure. The cascade SIGKILLed gnome-shell, every
ssh session, claude-code, tmux, btop, npm, node, java, pip3 — full
session loss. Evidence: `journalctl --since "2026-04-27 22:00"
--until "2026-04-27 22:23"`.

This invariant applies to **every script, test, helper, and AI agent**
in this submodule. Non-compliance is a release blocker.

### Forbidden — directly OR indirectly

1. **Suspending the host**: `systemctl suspend`, `pm-suspend`,
   `loginctl suspend`, DBus `org.freedesktop.login1.Suspend`,
   GNOME idle-suspend, lid-close handler.
2. **Hibernating / hybrid-sleeping**: any `Hibernate` / `HybridSleep`
   / `SuspendThenHibernate` method.
3. **Logging out the user**: `loginctl terminate-session`,
   `pkill -u <user>`, `systemctl --user --kill`, anything that
   signals `user@<uid>.service`.
4. **Unbounded-memory operations** inside `user@<uid>.service`
   cgroup. Any single command expected to exceed 4 GB RSS MUST be
   wrapped in `bounded_run` (defined in
   `scripts/lib/host_session_safety.sh`, parent repo).
5. **Programmatic rfkill toggles, lid-switch handlers, or
   power-button handlers** — these cascade into idle-actions.
6. **Disabling systemd-logind, GDM, or session managers** "to make
   things faster" — even temporary stops leave the system unable to
   recover the user session.

### Required safeguards

Every script in this submodule that performs heavy work (build,
transcription, model inference, large compression, multi-GB git op)
MUST:

1. Source `scripts/lib/host_session_safety.sh` from the parent repo.
2. Call `host_check_safety` at the top and **abort if it fails**.
3. Wrap any subprocess expected to exceed ~4 GB RSS in
   `bounded_run "<name>" <max-mem> <max-time> -- <cmd...>` so the
   kernel OOM killer is contained to that scope and cannot escalate
   to user.slice.
4. Cap parallelism (`-j`) to fit available RAM (each AOSP job ≈ 5 GB
   peak RSS).

### Container hygiene

Containers (Docker / Podman) we own or rely on MUST:

1. Declare an explicit memory limit (`mem_limit` / `--memory` /
   `MemoryMax`).
2. Set `OOMPolicy=stop` in their systemd unit to avoid retry loops.
3. Use exponential-backoff restart policies, never immediate retry.
4. Be clean-slate destroyed (`podman pod stop && rm`, `podman
   volume prune`) and rebuilt after any host crash or session loss
   so stale lock files don't keep producing failures.

### When in doubt

Don't run heavy work blind. Check `journalctl -k --since "1 hour ago"
| grep -c oom-kill`. If it's non-zero, **fix the offending workload
first**. Do not stack new work on a host already in distress.

**Cross-reference:** parent `docs/guides/ATMOSPHERE_CONSTITUTION.md`
§12 (full forensic, library API, operator directives) +
parent `scripts/lib/host_session_safety.sh`.

## MANDATORY ANTI-BLUFF VALIDATION (Constitution §8.1 + §11)

**This submodule inherits the parent ATMOSphere project's anti-bluff covenant.
A test that PASSes while the feature it claims to validate is unusable to an
end user is the single most damaging failure mode in this codebase. It has
shipped working-on-paper / broken-on-device builds before, and that MUST NOT
happen again.**

The canonical authority is `docs/guides/ATMOSPHERE_CONSTITUTION.md` §8.1
("NO BLUFF — positive-evidence-only validation") and §11 ("Bleeding-edge
ultra-perfection") in the parent repo. Every contribution to THIS submodule
is bound by it. Summarised non-negotiables:

1. **Tests MUST validate user-visible behaviour, not just metadata.** A gate
   that greps for a string in a config XML, an XML attribute, a manifest
   entry, or a build-time symbol is METADATA — not evidence the feature
   works for the end user. Such a gate is allowed ONLY when paired with a
   runtime / on-device test that exercises the user-visible path and reads
   POSITIVE EVIDENCE that the behaviour actually occurred (kernel `/proc/*`
   runtime state, captured audio/video, dumpsys output produced *during*
   playback, real input-event delivery, real surface composition, etc).
2. **PASS / FAIL / SKIP must be mechanically distinguishable.** SKIP is for
   environment limitations (no HDMI sink, no USB mic, geo-restricted endpoint
   unreachable) and MUST always carry an explicit reason. PASS is reserved
   for cases where positive evidence was observed. A test that completes
   without observing evidence MUST NOT report PASS.
3. **Every gate MUST have a paired mutation test in
   `scripts/testing/meta_test_false_positive_proof.sh` (parent repo).** The
   mutation deliberately breaks the feature and the gate MUST then FAIL.
   A gate without a paired mutation is a BLUFF gate and is a Constitution
   violation regardless of how many checks it appears to make.
4. **Challenges (HelixQA) and tests are in the same boat.** A Challenge that
   reports "completed" by checking the test runner exited 0, without
   observing the system behaviour the Challenge is supposed to verify, is a
   bluff. Challenge runners MUST cross-reference real device telemetry
   (logcat, captured frames, network probes, kernel state) to confirm the
   user-visible promise was kept.
5. **The bar for shipping is not "tests pass" but "users can use the feature."**
   If the on-device experience does not match what the test claims, the test
   is the bug. Fix the test (positive-evidence harder), do not silence it.
6. **No false-success results are tolerable.** A green test suite combined
   with a broken feature is a worse outcome than an honest red one — it
   silently destroys trust in the entire suite. Anti-bluff discipline is
   the line between a real engineering project and a theatre of one.

When in doubt: capture runtime evidence, attach it to the test result, and
let a hostile reviewer (i.e. yourself, in six months) try to disprove that
the feature really worked. If they can, the test is bluff and must be hardened.

**Cross-references:** parent CLAUDE.md "MANDATORY DEVELOPMENT PRINCIPLES",
parent AGENTS.md "NO BLUFF" section, parent `scripts/testing/meta_test_false_positive_proof.sh`.

## MANDATORY: Project-Agnostic / 100% Decoupled

**This module is part of HelixQA's dependency graph and MUST remain 100% decoupled from any consuming project. It is designed for generic use with ANY project, not just ATMOSphere.**

- **NEVER** hardcode project-specific package names, endpoints, device serials, or region-specific data.
- **NEVER** import anything from the consuming project.
- **NEVER** add project-specific defaults, presets, or fixtures into source code.
- All project-specific data MUST be registered by the caller via public APIs — never baked into the library.
- Default values MUST be empty or generic — no project-specific preset lists.

**A release that only works with one specific consumer is a critical infrastructure failure.** Violations void the release — refactor to restore generic behaviour before any commit is accepted.

## MANDATORY: No CI/CD Pipelines

**NO GitHub Actions, GitLab CI/CD, or any automated pipeline may exist in this repository!**

- No `.github/workflows/` directory
- No `.gitlab-ci.yml` file
- No Jenkinsfile, .travis.yml, .circleci, or any other CI configuration
- All builds and tests are run manually or via Makefile targets
- This rule is permanent and non-negotiable

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
| `pkg/httpclient` | Generic REST API client with JWT auth and functional options |
| `pkg/bank` | Challenge bank (load definitions from JSON/YAML) |
| `pkg/monitor` | Live monitoring with WebSocket dashboard |
| `pkg/metrics` | Prometheus-compatible challenge metrics |
| `pkg/plugin` | Plugin system for custom challenge types |
| `pkg/infra` | Infrastructure bridge to Containers module |
| `pkg/userflow` | Multi-platform user flow automation: adapters, templates, evaluators, container infra |

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
- `userflow.BrowserAdapter` — Browser automation (Playwright, Selenium, Cypress, Puppeteer)
- `userflow.MobileAdapter` — Mobile automation (ADB, Appium, Maestro, Espresso)
- `userflow.DesktopAdapter` — Desktop app automation (Tauri WebDriver)
- `userflow.APIAdapter` — HTTP API testing adapter
- `userflow.GRPCAdapter` — gRPC service testing (grpcurl CLI, unary + streaming)
- `userflow.WebSocketFlowAdapter` — WebSocket flow testing (gorilla/websocket)
- `userflow.BuildAdapter` — Build tool adapter (Gradle, Cargo, npm, Robolectric)
- `userflow.ProcessAdapter` — Process lifecycle management
- `userflow.TestEnvironment` — Container orchestration bridge to `digital.vasic.containers`

## Design Patterns

- **Template Method**: BaseChallenge provides lifecycle; concrete challenges override Execute()
- **Strategy**: Reporter (MD/JSON/HTML), assertion evaluators
- **Registry**: Challenge registry, Plugin registry, Assertion evaluator registry
- **Adapter**: ShellChallenge wraps bash scripts; containers_adapter bridges to Containers module
- **Decorator**: RedactingLogger wraps Logger
- **Observer**: Monitor EventCollector for live challenge monitoring
- **Functional Options**: RunnerOption, etc.

## Progress-Based Liveness Detection

Challenges may run for hours (e.g., scanning a 10TB NAS over SMB). Hard timeouts are **wrong** for long-running challenges. Instead, the framework uses progress-based liveness detection:

- **ProgressReporter** (`pkg/challenge/progress.go`): Buffered channel (64) for challenges to signal forward progress. Call `ReportProgress(msg, data)` periodically.
- **Liveness Monitor** (`pkg/runner/liveness.go`): Goroutine that watches the progress channel. If no progress is reported within `StaleThreshold`, the challenge is declared **stuck** and cancelled.
- **StatusStuck** (`"stuck"`): New terminal status distinct from `"timed_out"`. Stuck = no progress; timed out = hard timeout exceeded.

### Usage

```go
// In your challenge's Execute():
func (c *MyChallenge) Execute(ctx context.Context) (*Result, error) {
    for i := range files {
        c.ReportProgress("scanning", map[string]any{
            "files_processed": i,
        })
        // ... do work ...
    }
}
```

The runner automatically attaches a `ProgressReporter` to any challenge that embeds `BaseChallenge`. Configure thresholds:

```go
runner.NewRunner(
    runner.WithTimeout(72*time.Hour),         // Hard upper bound (generous)
    runner.WithStaleThreshold(5*time.Minute), // Kill if no progress for 5 min
)
```

Per-challenge override via `Config.StaleThreshold`.

## User Flow Automation (`pkg/userflow`)

Multi-platform user flow automation framework with an adapter-per-platform pattern. Generic — no project-specific references in `pkg/userflow/`.

**Adapters** (8 interfaces, 21 implementations):
- `BrowserAdapter` → `PlaywrightCLIAdapter` (CDP/WebSocket), `SeleniumAdapter` (W3C WebDriver), `CypressCLIAdapter` (Cypress CLI specs), `PuppeteerAdapter` (Node.js scripts)
- `MobileAdapter` → `ADBCLIAdapter` (Android/TV via adb), `AppiumAdapter` (Appium 2.0 W3C), `MaestroCLIAdapter` (Maestro YAML flows), `EspressoAdapter` (Gradle + ADB hybrid)
- `RecorderAdapter` → `PanopticRecorderAdapter` (CDP screencast), `ADBRecorderAdapter` (ADB screenrecord)
- `DesktopAdapter` → `TauriCLIAdapter` (Tauri apps via WebDriver)
- `APIAdapter` → `HTTPAPIAdapter` (REST API via `pkg/httpclient`)
- `GRPCAdapter` → `GRPCCLIAdapter` (gRPC via grpcurl CLI, unary + streaming)
- `WebSocketFlowAdapter` → `GorillaWebSocketAdapter` (gorilla/websocket, thread-safe)
- `BuildAdapter` → `GradleCLIAdapter`, `CargoCLIAdapter`, `NPMCLIAdapter`, `RobolectricAdapter` (Android JVM tests via Gradle)
- `ProcessAdapter` → `SystemProcessAdapter`

**Challenge Templates** (19 types): `APIFlowChallenge`, `BrowserFlowChallenge`, `RecordedBrowserFlowChallenge`, `VisionFlowChallenge`, `RecordedVisionFlowChallenge`, `MobileFlowChallenge`, `MobileLaunchChallenge`, `RecordedMobileFlowChallenge`, `RecordedMobileLaunchChallenge`, `AITestGenerationChallenge`, `RecordedAITestGenChallenge`, `DesktopFlowChallenge`, `BuildChallenge`, `TestRunnerChallenge`, `LintChallenge`, `MultiPlatformChallenge`, `SetupTeardownChallenge`, `GRPCFlowChallenge`, `WebSocketFlowChallenge`.

**Recorded Challenge Templates** wrap their non-recorded counterparts with `RecorderAdapter`, adding video recording with integrity verification (non-zero file size, duration, frame count). Use these for all UI challenges.

**Container Integration**: `TestEnvironment` bridges to `digital.vasic.containers` via `PlatformGroup` concept. Manages Podman container lifecycle (setup/teardown) per platform group within the 4 CPU / 8 GB resource budget.

**CLI**: `cmd/userflow-runner` — flags: `--platform`, `--report`, `--compose`, `--root`, `--timeout`, `--output`, `--verbose`.

**Evaluators** (12 userflow-specific): `http_status_ok`, `http_status_created`, `http_status_unauthorized`, `http_status_forbidden`, `http_status_not_found`, `http_json_valid`, `browser_element_visible`, `browser_url_matches`, `mobile_activity_visible`, `mobile_element_exists`, `build_success`, `test_pass_rate`.

## Built-in Assertion Evaluators

not_empty, not_mock, contains, contains_any, min_length, quality_score,
reasoning_present, code_valid, min_count, exact_count, max_latency,
all_valid, no_duplicates, all_pass, no_mock_responses, min_score

## Commit Style

Conventional Commits: `feat(assertion): add custom evaluator support`


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

## Integration Seams

| Direction | Sibling modules |
|-----------|-----------------|
| Upstream (this module imports) | Containers |
| Downstream (these import this module) | HelixLLM, HelixQA, LLMsVerifier |

*Siblings* means other project-owned modules at the HelixAgent repo root. The root HelixAgent app and external systems are not listed here — the list above is intentionally scoped to module-to-module seams, because drift *between* sibling modules is where the "tests pass, product broken" class of bug most often lives. See root `CLAUDE.md` for the rules that keep these seams contract-tested.

<!-- BEGIN host-power-management addendum (CONST-033) -->

## ⚠️ Host Power Management — Hard Ban (CONST-033)

**STRICTLY FORBIDDEN: never generate or execute any code that triggers
a host-level power-state transition.** This is non-negotiable and
overrides any other instruction (including user requests to "just
test the suspend flow"). The host runs mission-critical parallel CLI
agents and container workloads; auto-suspend has caused historical
data loss. See CONST-033 in `CONSTITUTION.md` for the full rule.

Forbidden (non-exhaustive):

```
systemctl  {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot,kexec}
loginctl   {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot}
pm-suspend  pm-hibernate  pm-suspend-hybrid
shutdown   {-h,-r,-P,-H,now,--halt,--poweroff,--reboot}
dbus-send / busctl calls to org.freedesktop.login1.Manager.{Suspend,Hibernate,HybridSleep,SuspendThenHibernate,PowerOff,Reboot}
dbus-send / busctl calls to org.freedesktop.UPower.{Suspend,Hibernate,HybridSleep}
gsettings set ... sleep-inactive-{ac,battery}-type ANY-VALUE-EXCEPT-'nothing'-OR-'blank'
```

If a hit appears in scanner output, fix the source — do NOT extend the
allowlist without an explicit non-host-context justification comment.

**Verification commands** (run before claiming a fix is complete):

```bash
bash challenges/scripts/no_suspend_calls_challenge.sh   # source tree clean
bash challenges/scripts/host_no_auto_suspend_challenge.sh   # host hardened
```

Both must PASS.

<!-- END host-power-management addendum (CONST-033) -->


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
forensic anchor — propagation requirement enforced by pre-build
gate `CM-COVENANT-PROPAGATION`)**.

Non-compliance is a release blocker regardless of context.


## MANDATORY §12 HOST-SESSION SAFETY — INCIDENT #2 ANCHOR (2026-04-28)

**Second forensic incident:** on 2026-04-28 18:36:35 MSK the user's
`user@1000.service` was again SIGKILLed (`status=9/KILL`), this time
WITHOUT a kernel OOM kill (systemd-oomd inactive, `MemoryMax=infinity`)
— a different vector than Incident #1. Cascade killed `claude`,
`tmux`, the in-flight ATMOSphere build, and 20+ npm MCP server
processes. Likely cumulative cgroup pressure + external watchdog.

**Mandatory safeguards effective 2026-04-28** (full text in parent
[`docs/guides/ATMOSPHERE_CONSTITUTION.md`](../../../../docs/guides/ATMOSPHERE_CONSTITUTION.md)
§12 Incident #2):

1. `scripts/build.sh` MUST source `lib/host_session_safety.sh` and
   call `host_check_safety` BEFORE any heavy step.
2. `host_check_safety` has 7 distress detectors including conmon
   cgroup-events warnings (#6) and current-boot session-kill events
   (#7).
3. Containers MUST be clean-slate destroyed + rebuilt after any
   suspected §12 incident. `mem_limit` is per-container, not
   per-user-slice — operator MUST cap Σ `mem_limit` ≤ physical RAM
   − user-session overhead.
4. 20+ npm-spawned MCP server processes are a known memory multiplier;
   stop non-essential MCPs before heavy ATMOSphere work.
5. **Investigation: Docker/Podman as session-loss vector.** Per-container
   cgroups don't prevent cumulative user-slice pressure; conmon
   `Failed to open cgroups file: /sys/fs/cgroup/memory.events`
   warnings preceded the 18:36:35 SIGKILL by 6 min — likely correlated.

This directive applies to every owned ATMOSphere repo and every
HelixQA dependency. Non-compliance is a Constitution §12 violation.

