#!/usr/bin/env bash
set -euo pipefail

script_dir="$(
  cd -- "$(dirname -- "${BASH_SOURCE[0]}")"
  pwd
)"
target_script="${script_dir}/ensure-oauth-login.sh"

if [[ ! -f "$target_script" ]]; then
  echo "missing ensure-oauth-login.sh: $target_script" >&2
  exit 1
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
export MOCK_SKILL_HOME_LOG="${tmp_dir}/calls.log"
export MOCK_SKILL_HOME_STATE="${tmp_dir}/authenticated"

skill-home() {
  printf '%s\n' "$*" >> "$MOCK_SKILL_HOME_LOG"

  if [[ "$1" == "login" && "${2:-}" == "--help" ]]; then
    echo "      --no-browser"
    echo "      --oauth-timeout duration"
    return 0
  fi

  if [[ "$1" == "whoami" ]]; then
    if [[ -f "$MOCK_SKILL_HOME_STATE" && -z "${SKILL_HOME_API_KEY:-}" ]]; then
      echo "用户名: tester"
      return 0
    fi
    return 1
  fi

  if [[ "$1" == "login" ]]; then
    touch "$MOCK_SKILL_HOME_STATE"
    return 0
  fi

  return 1
}
export -f skill-home

# A valid existing login must be reused without starting OAuth.
touch "$MOCK_SKILL_HOME_STATE"
: > "$MOCK_SKILL_HOME_LOG"
output="$(bash "$target_script")"
grep -q '用户名: tester' <<< "$output"
if grep -q '^login' "$MOCK_SKILL_HOME_LOG"; then
  echo "existing login unexpectedly started OAuth" >&2
  exit 1
fi

# An invalid environment key must be ignored inside the helper; OAuth should
# receive all requested options and whoami should succeed afterward.
rm -f "$MOCK_SKILL_HOME_STATE"
: > "$MOCK_SKILL_HOME_LOG"
export SKILL_HOME_API_KEY="invalid-test-key"
output="$(bash "$target_script" --server https://registry.example.com --no-browser --oauth-timeout 15m)"
unset SKILL_HOME_API_KEY

grep -q '用户名: tester' <<< "$output"
grep -q '^login --help$' "$MOCK_SKILL_HOME_LOG"
grep -q '^login --server https://registry.example.com --no-browser --oauth-timeout 15m$' "$MOCK_SKILL_HOME_LOG"
grep -q '^whoami$' "$MOCK_SKILL_HOME_LOG"

# With no server override, retry a valid persisted credential after removing an
# invalid environment key instead of starting an unnecessary OAuth flow.
touch "$MOCK_SKILL_HOME_STATE"
: > "$MOCK_SKILL_HOME_LOG"
export SKILL_HOME_API_KEY="invalid-test-key"
output="$(bash "$target_script")"
unset SKILL_HOME_API_KEY

grep -q '用户名: tester' <<< "$output"
if grep -q '^login' "$MOCK_SKILL_HOME_LOG"; then
  echo "persisted login unexpectedly started OAuth after ignoring an invalid environment key" >&2
  exit 1
fi

echo "test-ensure-oauth-login: ok"
