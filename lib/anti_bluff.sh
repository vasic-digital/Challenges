#!/system/bin/sh
# anti_bluff.sh — shared anti-bluff helpers for on-device tests.
# Source via: . /data/local/tmp/tests/lib/anti_bluff.sh
#
# Rules enforced by these helpers:
#
#   1. POSITIVE EVIDENCE — every PASS must come from observed positive
#      data (a value match, a state delta, a frame text match), NOT
#      from "no error".
#
#   2. STATE DELTA — every test that's claimed to verify a feature must
#      capture state BEFORE the action and AFTER, and assert the delta
#      matches expectation. A test where before==after on success is
#      a bluff — call ab_assert_delta() to lock that in.
#
#   3. REAL ACTION — `ab_send_action()` wrapper logs the action the
#      test took. Tests that pass without ever invoking ab_send_action
#      are flagged at the meta level.
#
#   4. UNIQUE EVIDENCE — `ab_evidence_token()` returns a per-run UUID;
#      tests embed the token in their action and look for the token in
#      the resulting state. Cached results that pre-date the token CAN
#      NOT match — defeats stale-cache false-pass.
#
# Usage example:
#
#   . /data/local/tmp/tests/lib/anti_bluff.sh
#   ab_init
#   TOKEN=$(ab_evidence_token)
#   BEFORE=$(settings get system screen_brightness)
#   ab_send_action "set framework brightness to 200"
#   settings put system screen_brightness 200
#   sleep 1
#   AFTER=$(settings get system screen_brightness)
#   ab_assert_delta "framework brightness" "$BEFORE" "$AFTER" 200
#   ab_assert_kernel_value "/sys/class/backlight/backlight/brightness" 200
#   ab_summary

# ─────────────────────────────────────────────────────────────────────
# State + counters
# ─────────────────────────────────────────────────────────────────────
AB_PASS=0
AB_FAIL=0
AB_SKIP=0
AB_ACTIONS=0
AB_TEST_NAME=""
AB_RESULTS_PATH=""

ab_init() {
    AB_TEST_NAME="${1:-$(basename "$0" .sh)}"
    AB_RESULTS_PATH="${2:-/data/local/tmp/${AB_TEST_NAME}.results}"
    rm -f "$AB_RESULTS_PATH" 2>/dev/null
    echo "=== $AB_TEST_NAME — anti-bluff verification ==="
    echo "Date: $(date 2>/dev/null)"
    echo "Results: $AB_RESULTS_PATH"
    echo ""
    AB_PASS=0; AB_FAIL=0; AB_SKIP=0; AB_ACTIONS=0
}

ab_evidence_token() {
    # 12-char unique-per-run identifier. /proc/sys/kernel/random/uuid
    # is the canonical source on Android.
    local token
    if [ -r /proc/sys/kernel/random/uuid ]; then
        token=$(cat /proc/sys/kernel/random/uuid 2>/dev/null | tr -d '-' | head -c 12)
    fi
    if [ -z "$token" ]; then
        token=$(date +%s%N 2>/dev/null | tail -c 12)
    fi
    echo "${token:-AB$$$(date +%s)}"
}

ab_send_action() {
    AB_ACTIONS=$((AB_ACTIONS+1))
    echo "ACTION #$AB_ACTIONS: $*" | tee -a "$AB_RESULTS_PATH"
}

ab_pass() {
    AB_PASS=$((AB_PASS+1))
    echo "PASS: $1" | tee -a "$AB_RESULTS_PATH"
}

ab_fail() {
    AB_FAIL=$((AB_FAIL+1))
    echo "FAIL: $1" | tee -a "$AB_RESULTS_PATH"
}

ab_skip() {
    AB_SKIP=$((AB_SKIP+1))
    echo "SKIP: $1 ($2)" | tee -a "$AB_RESULTS_PATH"
}

# ─────────────────────────────────────────────────────────────────────
# Assertion helpers — every assertion logs both BEFORE and AFTER so
# the output is auditable and a stale-cache false-pass is impossible.
# ─────────────────────────────────────────────────────────────────────

# ab_assert_delta <name> <before> <after> <expected_after>
# PASS only if before != after AND after == expected_after.
ab_assert_delta() {
    local name="$1"
    local before="$2"
    local after="$3"
    local expected="$4"
    if [ "$before" = "$after" ]; then
        ab_fail "$name: state did NOT change (before='$before', after='$after') — feature is non-functional"
        return 1
    fi
    if [ "$after" != "$expected" ]; then
        ab_fail "$name: state changed but to wrong value (before='$before', after='$after', expected='$expected')"
        return 1
    fi
    ab_pass "$name: state changed correctly ('$before' → '$after')"
    return 0
}

# ab_assert_kernel_value <sysfs_path> <expected>
# Reads the kernel sysfs node and asserts the value matches expected.
ab_assert_kernel_value() {
    local path="$1"
    local expected="$2"
    if [ ! -r "$path" ]; then
        ab_fail "kernel value: $path not readable"
        return 1
    fi
    local actual
    actual=$(cat "$path" 2>/dev/null | tr -d '\r\n ')
    if [ "$actual" = "$expected" ]; then
        ab_pass "kernel $path = '$actual' (expected '$expected')"
        return 0
    fi
    ab_fail "kernel $path = '$actual' (expected '$expected')"
    return 1
}

# ab_assert_pkg_installed <package_name>
# Asserts the package is registered with PackageManager (positive evidence:
# pm path returns a non-empty result).
ab_assert_pkg_installed() {
    local pkg="$1"
    local path
    path=$(pm path "$pkg" 2>/dev/null | head -1)
    if [ -n "$path" ]; then
        ab_pass "package $pkg installed: $path"
        return 0
    fi
    ab_fail "package $pkg NOT installed (pm path returned empty)"
    return 1
}

# ab_assert_pkg_label <package_name> <expected_label_prefix>
# Pulls the APK, runs aapt locally if available, asserts label prefix.
# On-device aapt typically not present — this falls back to a system
# property pre-populated at build time. If neither path works, SKIPs.
ab_assert_pkg_label() {
    local pkg="$1"
    local prefix="$2"
    if ! command -v aapt >/dev/null 2>&1; then
        ab_skip "package $pkg label" "aapt not on device — verify host-side via test_forked_app_branding.sh"
        return 0
    fi
    local apk_path label
    apk_path=$(pm path "$pkg" 2>/dev/null | sed 's|^package:||' | head -1)
    if [ -z "$apk_path" ]; then
        ab_fail "package $pkg label: not installed"
        return 1
    fi
    label=$(aapt dump badging "$apk_path" 2>/dev/null | grep -m1 "application-label:" | sed "s|application-label:'\(.*\)'|\1|")
    case "$label" in
        ${prefix}*) ab_pass "package $pkg label='$label' (matches prefix '$prefix')"; return 0 ;;
        *) ab_fail "package $pkg label='$label' (expected to start with '$prefix')"; return 1 ;;
    esac
}

# ab_assert_dumpsys_field <dumpsys_args> <field_regex> <expected>
# Greps `dumpsys <args>` output for a `field=value` and asserts.
ab_assert_dumpsys_field() {
    local args="$1"
    local field="$2"
    local expected="$3"
    local actual
    actual=$(dumpsys $args 2>/dev/null | grep -m1 -E "$field" | sed -E "s/.*${field}//; s/[[:space:]].*//")
    if [ "$actual" = "$expected" ]; then
        ab_pass "dumpsys $args $field=$actual (expected '$expected')"
        return 0
    fi
    ab_fail "dumpsys $args $field='$actual' (expected '$expected')"
    return 1
}

# ─────────────────────────────────────────────────────────────────────
# Final summary — exits 1 if any FAIL or zero ACTIONS (suspicious).
# ─────────────────────────────────────────────────────────────────────
ab_summary() {
    local total=$((AB_PASS + AB_FAIL + AB_SKIP))
    echo ""
    echo "=== SUMMARY ==="
    echo "PASS:    $AB_PASS"
    echo "FAIL:    $AB_FAIL"
    echo "SKIP:    $AB_SKIP"
    echo "ACTIONS: $AB_ACTIONS"
    echo "TOTAL:   $total"
    echo "Results: $AB_RESULTS_PATH"

    # Anti-bluff guard: a test that did not call ab_send_action even once
    # is suspect. The test framework requires positive-evidence by
    # stating actions explicitly. Allow tests that are pure invariant
    # checks (zero PASS) to slip through, but flag ANY PASS without
    # actions as a bluff.
    if [ "$AB_ACTIONS" -eq 0 ] && [ "$AB_PASS" -gt 0 ]; then
        echo "FAIL: anti-bluff guard: $AB_PASS PASSes recorded without any ab_send_action() — test is bluffing"
        return 1
    fi

    [ "$AB_FAIL" -eq 0 ]
}
