#!/usr/bin/env bash
set -euo pipefail

script_dir="$(
  cd -- "$(dirname -- "${BASH_SOURCE[0]}")"
  pwd
)"

expand_path() {
  local path="$1"

  case "$path" in
    "~")
      printf '%s\n' "$HOME"
      ;;
    "~/"*)
      printf '%s\n' "${HOME}/${path#~/}"
      ;;
    *)
      printf '%s\n' "$path"
      ;;
  esac
}

resolve_codex_skills_dir() {
  if [[ -n "${SKILL_HOME_CODEX_SKILLS_DIR:-}" ]]; then
    expand_path "$SKILL_HOME_CODEX_SKILLS_DIR"
    return
  fi

  config_path="${SKILL_HOME_CONFIG:-${HOME}/.config/skill-home/config.yaml}"
  if [[ -f "$config_path" ]]; then
    parsed="$(
      awk '
        /^[[:space:]]*ide:[[:space:]]*$/ {
          in_ide=1
          next
        }
        in_ide && /^[[:space:]]*[A-Za-z0-9_-]+:[[:space:]]*$/ {
          line=$0
          gsub(/^[[:space:]]+|:[[:space:]]*$/, "", line)
          in_codex=(line=="codex")
          next
        }
        in_codex && /^[[:space:]]*global_path:[[:space:]]*/ {
          sub(/^[[:space:]]*global_path:[[:space:]]*/, "", $0)
          gsub(/["'\'']/, "", $0)
          print $0
          exit
        }
      ' "$config_path"
    )"
    if [[ -n "$parsed" ]]; then
      expand_path "$parsed"
      return
    fi
  fi

  if [[ -n "${CODEX_HOME:-}" ]]; then
    printf '%s\n' "$(expand_path "$CODEX_HOME")/skills"
    return
  fi

  if [[ -d "${HOME}/.codex" ]]; then
    printf '%s\n' "${HOME}/.codex/skills"
    return
  fi

  if command -v wslpath >/dev/null 2>&1 && [[ -n "${USERPROFILE:-}" ]]; then
    win_home="$(wslpath "$USERPROFILE")"
    if [[ -d "${win_home}/.codex" ]]; then
      printf '%s\n' "${win_home}/.codex/skills"
      return
    fi
  fi

  if [[ -d "/mnt/c/Users/${USER}/.codex" ]]; then
    printf '%s\n' "/mnt/c/Users/${USER}/.codex/skills"
    return
  fi

  printf '%s\n' "${HOME}/.codex/skills"
}

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

"$script_dir/bootstrap-cli.sh"

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
