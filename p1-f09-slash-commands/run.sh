#!/usr/bin/env bash
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
cd "$ROOT/HelixCode"

echo "==> build F09 challenge harness"
HARNESS_BIN="$(mktemp -d)/p1f09_challenge"
go build -o "$HARNESS_BIN" ./tests/integration/cmd/p1f09_challenge

echo "==> run harness"
"$HARNESS_BIN"

echo "==> anti-bluff smoke on F09-affected code"
if grep -rn "simulated\|for now\|TODO implement\|placeholder" \
    internal/commands/markdown_commands.go \
    internal/commands/markdown_watcher.go \
    internal/commands/commands_command.go \
    cmd/cli/commands_cmd.go; then
    echo "BLUFF FOUND" >&2
    exit 1
fi
echo "clean"

echo "==> cross-compile linux"
go build ./cmd/cli/... ./internal/commands/

echo "==> P1-F09 challenge PASS"
