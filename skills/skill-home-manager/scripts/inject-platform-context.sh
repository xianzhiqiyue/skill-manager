#!/usr/bin/env bash
set -euo pipefail

script_dir="$(
  cd -- "$(dirname -- "${BASH_SOURCE[0]}")"
  pwd
)"

# shellcheck source=/dev/null
source "${script_dir}/common.sh"

usage() {
  cat <<'EOF'
Usage: inject-platform-context.sh <platform|auto> <installed-skill-dir> [target-scope] [target-path] [install-mode]

Persist platform context into <installed-skill-dir>/.skill-home/platform-context.json.
Use platform=auto to infer from the installed path or Xigua package manifest.
EOF
}

if [[ $# -lt 2 || $# -gt 5 ]]; then
  usage
  exit 1
fi

platform="$1"
installed_skill_dir="$2"
target_scope="${3:-unknown}"
target_path="${4:-}"
install_mode="${5:-unknown}"

write_platform_context "$platform" "$installed_skill_dir" "$target_scope" "$target_path" "$install_mode"
