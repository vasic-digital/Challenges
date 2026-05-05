#!/usr/bin/env bash
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
cd "$ROOT/HelixCode"

echo "==> build F10 challenge harness"
HARNESS_BIN="$(mktemp -d)/p1f10_challenge"
go build -o "$HARNESS_BIN" ./tests/integration/cmd/p1f10_challenge

echo "==> run harness"
"$HARNESS_BIN"

echo "==> anti-bluff smoke on F10-affected code"
if grep -rn "simulated\|for now\|TODO implement\|placeholder" \
    internal/commands/markdown_skills.go \
    internal/commands/skills_watcher.go \
    internal/commands/skills_command.go \
    cmd/cli/skills_cmd.go \
    internal/agent/skill_dispatcher.go; then
    echo "BLUFF FOUND" >&2
    exit 1
fi
echo "clean"

echo "==> cross-compile linux"
go build ./cmd/cli/... ./internal/commands/ ./internal/agent/

echo "==> P1-F10 challenge PASS"
