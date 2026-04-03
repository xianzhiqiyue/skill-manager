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
Usage: install-to-codex.sh <skill-path>

Validate a local skill, sync it into the resolved Codex global skills directory
using mirror mode, then probe compatible candidate paths to verify the install.
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
skill-home sync "$skill_path" --ide codex --global --mode mirror

install_dir="$(resolve_installed_codex_skill_dir "$skill_name" || true)"

if [[ -z "$install_dir" ]]; then
  echo "Install verification failed: unable to locate ${skill_name}/SKILL.md in compatible Codex skill directories" >&2
  echo "Checked:" >&2
  while IFS= read -r candidate; do
    echo "  - ${candidate%/}/$skill_name" >&2
  done < <(list_codex_skills_dir_candidates)
  exit 1
fi

find "$install_dir" -maxdepth 3 -type f | sort
