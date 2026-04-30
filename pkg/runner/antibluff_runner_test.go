// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Anti-bluff runner-integration tests — verify that
// CHALLENGE_ANTIBLUFF_STRICT=1 actually downgrades a bluff
// Status=Passed result to StatusFailed at runner integration time.
// Constitution §11.4 — User mandate 2026-04-28.

package runner

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/challenge"
)

// TestStrictMode_BluffPassDowngraded confirms that with strict mode
// enabled, a bluff result (Status=Passed but no RecordedActions, no
// passing assertions) is downgraded to StatusFailed and the Error
// field surfaces ErrBluffPass.
func TestStrictMode_BluffPassDowngraded(t *testing.T) {
	t.Setenv("CHALLENGE_ANTIBLUFF_STRICT", "1")

	// newStub returns a passing-Status stub WITHOUT RecordedActions
	// and WITHOUT Assertions — the canonical bluff pattern.
	s := newStub("bluff-1")
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	result, err := r.Run(context.Background(), "bluff-1", challenge.NewConfig("bluff-1"))
	require.NoError(t, err)
	require.NotNil(t, result)

	if result.Status == challenge.StatusPassed {
		t.Fatalf("expected status downgraded from Passed; got Passed (the validator did not engage)")
	}
	if result.Status != challenge.StatusFailed {
		t.Fatalf("expected StatusFailed (the canonical bluff downgrade); got %q", result.Status)
	}
	if !strings.Contains(result.Error, "bluff") {
		t.Fatalf("expected Error to mention 'bluff'; got %q", result.Error)
	}
}

// TestStrictMode_OffPreservesLegacyBehavior — the gate is OPT-IN.
// When the env var is unset, a bluff result still propagates as
// StatusPassed. This is the backward-compat lane that lets existing
// fixtures keep passing while the per-test ratchet adds RecordAction
// calls (parallel to the Bash side's Phase 22.0–22.3 conversion).
func TestStrictMode_OffPreservesLegacyBehavior(t *testing.T) {
	// Explicitly clear (Setenv with "" doesn't unset; the test
	// runner's t.Setenv guarantees restoration after the test).
	t.Setenv("CHALLENGE_ANTIBLUFF_STRICT", "")

	s := newStub("bluff-2")
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	result, err := r.Run(context.Background(), "bluff-2", challenge.NewConfig("bluff-2"))
	require.NoError(t, err)
	require.NotNil(t, result)

	if result.Status != challenge.StatusPassed {
		t.Fatalf("expected StatusPassed in legacy (env-unset) mode; got %q (Error: %q)",
			result.Status, result.Error)
	}
}
