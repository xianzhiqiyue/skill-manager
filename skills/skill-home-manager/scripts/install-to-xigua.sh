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
Usage: install-to-xigua.sh <skill-path>

Validate a local skill, sync it into the resolved Xigua global skills directory
using mirror mode, then verify the Xigua package layout.
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

skill-home validate "$skill_path"
skill-home sync "$skill_path" --ide xigua --global --mode mirror

install_dir="$(resolve_installed_xigua_skill_dir "$skill_name" || true)"

if [[ -z "$install_dir" ]]; then
  echo "Install verification failed: unable to locate ${skill_name}/SKILL.md in compatible Xigua skill directories" >&2
  echo "Checked:" >&2
  while IFS= read -r candidate; do
    echo "  - ${candidate%/}/$skill_name" >&2
  done < <(list_xigua_skills_dir_candidates)
  exit 1
fi

if [[ ! -f "$install_dir/skill.json" ]]; then
  echo "Install verification failed: missing Xigua package manifest: $install_dir/skill.json" >&2
  exit 1
fi

write_platform_context "xigua" "$install_dir" "global" "$(dirname "$install_dir")" "mirror" >/dev/null
find "$install_dir" -maxdepth 3 -type f | sort
