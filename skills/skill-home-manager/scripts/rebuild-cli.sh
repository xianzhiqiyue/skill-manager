#!/usr/bin/env bash
set -euo pipefail

repo_root="${REPO_ROOT:-/home/zhuyue/code/skill-manager}"
cli_dir="$repo_root/skill-home-cli"
install_binary=1

usage() {
  cat <<'EOF'
Usage: rebuild-cli.sh [--build-only]

Rebuild the local skill-home CLI from source. By default this also installs the
new binary to /usr/local/bin/skill-home.
EOF
}

if [[ $# -gt 1 ]]; then
  usage
  exit 1
fi

if [[ $# -eq 1 ]]; then
  case "$1" in
    --build-only)
      install_binary=0
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
fi

if [[ ! -d "$cli_dir" ]]; then
  echo "CLI source directory not found: $cli_dir" >&2
  exit 1
fi

cd "$cli_dir"

go test ./...
make build

if [[ "$install_binary" -eq 1 ]]; then
  sudo install -m 0755 "$cli_dir/bin/skill-home" /usr/local/bin/skill-home
fi

skill-home version
