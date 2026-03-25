#!/usr/bin/env bash
set -euo pipefail

repo_root="${REPO_ROOT:-/home/zhuyue/code/skill-manager}"
cli_dir="$repo_root/skill-home-cli"
local_binary="$cli_dir/bin/skill-home"
user_bin_dir="${HOME}/.local/bin"
user_install_target="$user_bin_dir/skill-home"
system_install_target="/usr/local/bin/skill-home"
build_only=0
force_rebuild=0
skip_tests=0
system_install=0

usage() {
  cat <<'EOF'
Usage: bootstrap-cli.sh [--build-only] [--force-rebuild] [--skip-tests]

Ensure the local skill-home CLI is available. If `skill-home` is missing, this
script will:
  1. Install Go with apt-get when `go` is not available
  2. Build the CLI from the local repository source
  3. Install it to ~/.local/bin/skill-home by default

Flags:
  --build-only      Build the CLI but do not install it into PATH
  --force-rebuild   Rebuild even if `skill-home` already exists in PATH
  --skip-tests      Skip `go test ./...` before building
  --system          Install to /usr/local/bin/skill-home instead of ~/.local/bin
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --build-only)
      build_only=1
      shift
      ;;
    --force-rebuild)
      force_rebuild=1
      shift
      ;;
    --skip-tests)
      skip_tests=1
      shift
      ;;
    --system)
      system_install=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage
      exit 1
      ;;
  esac
done

if [[ ! -d "$cli_dir" ]]; then
  echo "CLI source directory not found: $cli_dir" >&2
  exit 1
fi

if [[ "$force_rebuild" -eq 0 ]] && command -v skill-home >/dev/null 2>&1; then
  existing_cli="$(command -v skill-home)"
  if [[ "$existing_cli" == "$user_install_target" || "$existing_cli" == "$system_install_target" || "$existing_cli" == "$local_binary" ]]; then
    version_output="$("$existing_cli" version 2>/dev/null || true)"
    if grep -q 'Version:[[:space:]]\+zip-archive' <<<"$version_output"; then
      echo "skill-home at $existing_cli looks like a packaged/stale build. Rebuilding from source..."
    else
      echo "skill-home already available in PATH: $existing_cli"
      printf '%s\n' "$version_output"
      exit 0
    fi
  fi

  echo "skill-home currently resolves to $existing_cli, not the repo-managed CLI. Rebuilding from source..."
fi

if ! command -v go >/dev/null 2>&1; then
  if [[ -x /usr/bin/apt-get || -x /bin/apt-get ]]; then
    echo "Go not found. Installing golang-go with apt-get..."
    sudo apt-get update
    sudo apt-get install -y golang-go
  else
    echo "Go is not installed and apt-get is unavailable. Install Go manually first." >&2
    exit 1
  fi
fi

mkdir -p "$cli_dir/bin"
cd "$cli_dir"

if [[ "$skip_tests" -eq 0 ]]; then
  go test ./...
fi

go build -o "$local_binary" ./cmd/skill-home

if [[ "$build_only" -eq 1 ]]; then
  "$local_binary" version
  exit 0
fi

if [[ "$system_install" -eq 1 ]]; then
  sudo install -m 0755 "$local_binary" "$system_install_target"
  "$system_install_target" version
  exit 0
fi

mkdir -p "$user_bin_dir"
install -m 0755 "$local_binary" "$user_install_target"
hash -r
skill-home version
