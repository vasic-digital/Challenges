---
schema_version: 1
constitution_rule: CONST-035
last_audit: 2026-05-01
---

# Behavior Anchor Manifest — Challenges

Every row is a user-facing capability and the single anchor test that
proves it works end-to-end. See CONST-035 in `CONSTITUTION.md`.

## Status legend

- `active` — anchor exists and is callable; capability is verified.
- `pending-anchor` — capability declared, anchor test does not yet
  exist. Listed in `challenges/baselines/bluff-baseline.txt` Section 3.
  Reducing this state is the work of campaign sub-project 4.
- `retired` — capability removed; row kept for history.

## Path format

For Go tests: `<path>.go::<TestFuncName>`. The challenge verifier
greps for `func <TestFuncName>\b` in the file.

## Capabilities

| id | layer | capability | anchor_test_path | verifies | status |
|----|-------|------------|------------------|----------|--------|
| CAP-001 | submodule:Challenges | Run a registered challenge end-to-end via DefaultRunner | pkg/runner/runner_test.go::TestDefaultRunner_Run_Success | Runner.Run() executes a registered challenge and returns a Result with status=passed | active |
| CAP-002 | submodule:Challenges | Register a challenge in DefaultRegistry without collisions | pkg/registry/registry_test.go::TestDefaultRegistry_Register_Success | Registry.Register() accepts a fresh challenge ID and returns no error | active |
| CAP-003 | submodule:Challenges | Evaluate built-in not_empty assertion | pkg/assertion/builtin_test.go::TestEvaluateNotEmpty | NotEmpty evaluator returns pass when given non-empty value, fail when empty | active |
| CAP-004 | submodule:Challenges | Generate Markdown report from challenge results | pkg/report/markdown_test.go::TestMarkdownReporter_GenerateReport_Content | MarkdownReporter.GenerateReport() produces valid Markdown with all results | active |
| CAP-005 | submodule:Challenges | Load challenge bank from JSON or YAML file | pkg/bank/bank_test.go::TestBank_LoadFile | Bank.LoadFile() parses challenge definitions and registers them | active |
| CAP-006 | submodule:Challenges | Construct ShellChallenge that wraps a bash script | pkg/challenge/shell_test.go::TestShellChallenge_NewShellChallenge | ShellChallenge constructor returns a configured Challenge with the script bound | active |
| CAP-007 | submodule:Challenges | Liveness monitor handles nil progress reporter without panicking | pkg/runner/liveness_test.go::TestLivenessMonitor_NilProgress_NoOp | LivenessMonitor with nil progress channel is a safe no-op | active |
| CAP-008 | submodule:Challenges | Construct API HTTP client with default config | pkg/httpclient/client_test.go::TestNewAPIClient_Defaults | NewAPIClient() returns a usable client with sensible defaults | active |
| CAP-009 | submodule:Challenges | Validate challenge result requires positive evidence (anti-bluff metatest) | pkg/challenge/antibluff_test.go::TestValidate_PassWithEvidence | Validate() accepts pass results with non-empty evidence; rejects metadata-only passes | active |
| CAP-010 | submodule:Challenges | userflow-runner CLI resolves "all" platform target to every registered platform key | cmd/userflow-runner/main_test.go::TestResolveGroups_AllPlatformExpandsEveryKey | CLI flag parsing produces the full platform set when --platform=all | active |

(More capabilities — runner-parallel, runner-pipeline, plugin loading,
infra/containers bridge, websocket monitor — populated in subsequent
iterations of sub-project 3.)
