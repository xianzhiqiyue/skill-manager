#!/usr/bin/env bash
set -euo pipefail

script_dir="$(
  cd -- "$(dirname -- "${BASH_SOURCE[0]}")"
  pwd
)"

default_output_dir="$PWD"
if [[ -d "$PWD/skills" ]]; then
  default_output_dir="$PWD/skills"
fi

usage() {
  cat <<'EOF'
Usage: create-local-skill.sh <skill-name> [description] [output-dir]

Create a new local skill directory by calling `skill-home init`, patch the
generated description if one is provided, and validate the result.
EOF
}

if [[ $# -lt 1 || $# -gt 3 ]]; then
  usage
  exit 1
fi

"$script_dir/bootstrap-cli.sh"

skill_name="$1"
description="${2:-}"
output_dir="${3:-$default_output_dir}"
skill_dir="$output_dir/$skill_name"
skill_file="$skill_dir/SKILL.md"

if [[ -e "$skill_dir" ]]; then
  echo "Target skill directory already exists: $skill_dir" >&2
  exit 1
fi

mkdir -p "$output_dir"
skill-home init "$skill_name" --output "$output_dir"

if [[ -n "$description" ]]; then
  escaped_description="$(printf '%s' "$description" | sed 's/[\/&]/\\&/g')"
  sed -i "0,/^description: .*/s//description: $escaped_description/" "$skill_file"
fi

skill-home validate "$skill_dir"
printf 'Created skill: %s\n' "$skill_dir"
