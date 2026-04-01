#!/usr/bin/env bash
set -euo pipefail

script_dir="$(
  cd -- "$(dirname -- "${BASH_SOURCE[0]}")"
  pwd
)"

# shellcheck source=/dev/null
source "${script_dir}/common.sh"

codex_skills_dir="$(resolve_codex_skills_dir)"

usage() {
  cat <<'EOF'
Usage: install-to-codex.sh <skill-path>

Validate a local skill and install it into the Codex global skills directory
using mirror mode, then verify that the installed SKILL.md exists.
EOF
}

if [[ $# -ne 1 ]]; then
  usage
  exit 1
fi

bash "$script_dir/bootstrap-cli.sh"

skill_path="$1"
skill_file="$skill_path/SKILL.md"

if [[ ! -f "$skill_file" ]]; then
  echo "SKILL.md not found: $skill_file" >&2
  exit 1
fi

skill_name="$(
  sed -n 's/^name:[[:space:]]*//p' "$skill_file" \
    | head -n 1 \
    | tr -d '"' \
    | tr -d "'"
)"

if [[ -z "$skill_name" ]]; then
  echo "Failed to parse skill name from $skill_file" >&2
  exit 1
fi

install_dir="$codex_skills_dir/$skill_name"

skill-home validate "$skill_path"
skill-home sync "$skill_path" --ide codex --global --mode mirror

if [[ ! -f "$install_dir/SKILL.md" ]]; then
  echo "Install verification failed: $install_dir/SKILL.md was not created" >&2
  exit 1
fi

find "$install_dir" -maxdepth 3 -type f | sort
