#!/usr/bin/env bash

set -euo pipefail

skill_home_manager_script_dir() {
  cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd
}

expand_path() {
  local path="${1:-}"

  case "$path" in
    "")
      printf '\n'
      ;;
    \~)
      printf '%s\n' "$HOME"
      ;;
    \~/*)
      printf '%s\n' "${HOME}/${path#\~/}"
      ;;
    *)
      printf '%s\n' "$path"
      ;;
  esac
}

parse_codex_global_path_from_config() {
  local config_path="$1"

  if [[ ! -f "$config_path" ]]; then
    return 1
  fi

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
}

resolve_codex_skills_dir() {
  local parsed=""
  local config_path=""
  local win_home=""

  if [[ -n "${SKILL_HOME_CODEX_SKILLS_DIR:-}" ]]; then
    expand_path "$SKILL_HOME_CODEX_SKILLS_DIR"
    return
  fi

  config_path="${SKILL_HOME_CONFIG:-${HOME}/.config/skill-home/config.yaml}"
  parsed="$(parse_codex_global_path_from_config "$config_path" || true)"
  if [[ -n "$parsed" ]]; then
    expand_path "$parsed"
    return
  fi

  if [[ -n "${CODEX_HOME:-}" ]]; then
    printf '%s\n' "$(expand_path "$CODEX_HOME")/skills"
    return
  fi

  if [[ -d "${HOME}/.codex" ]]; then
    printf '%s\n' "${HOME}/.codex/skills"
    return
  fi

  if [[ -d "${HOME}/.agents" ]]; then
    printf '%s\n' "${HOME}/.agents/skills"
    return
  fi

  if command -v wslpath >/dev/null 2>&1 && [[ -n "${USERPROFILE:-}" ]]; then
    win_home="$(wslpath "$USERPROFILE")"
    if [[ -d "${win_home}/.codex" ]]; then
      printf '%s\n' "${win_home}/.codex/skills"
      return
    fi
    if [[ -d "${win_home}/.agents" ]]; then
      printf '%s\n' "${win_home}/.agents/skills"
      return
    fi
  fi

  if [[ -d "/mnt/c/Users/${USER}/.codex" ]]; then
    printf '%s\n' "/mnt/c/Users/${USER}/.codex/skills"
    return
  fi

  if [[ -d "/mnt/c/Users/${USER}/.agents" ]]; then
    printf '%s\n' "/mnt/c/Users/${USER}/.agents/skills"
    return
  fi

  printf '%s\n' "${HOME}/.codex/skills"
}

is_skill_home_manager_root() {
  local candidate="${1:-}"

  if [[ -z "$candidate" || ! -f "$candidate/SKILL.md" ]]; then
    return 1
  fi

  grep -Eq '^name:[[:space:]]*skill-home-manager[[:space:]]*$' "$candidate/SKILL.md"
}

resolve_skill_home_manager_root() {
  local configured_skills_dir=""
  local candidate=""
  local script_dir=""
  local script_root=""
  local win_home=""

  if [[ -n "${SKILL_HOME_MANAGER_ROOT:-}" ]]; then
    candidate="$(expand_path "$SKILL_HOME_MANAGER_ROOT")"
    if is_skill_home_manager_root "$candidate"; then
      printf '%s\n' "$candidate"
      return
    fi
  fi

  configured_skills_dir="$(resolve_codex_skills_dir)"
  candidate="${configured_skills_dir%/}/skill-home-manager"
  if is_skill_home_manager_root "$candidate"; then
    printf '%s\n' "$candidate"
    return
  fi

  for candidate in \
    "${HOME}/.codex/skills/skill-home-manager" \
    "${HOME}/.agents/skills/skill-home-manager"
  do
    if is_skill_home_manager_root "$candidate"; then
      printf '%s\n' "$candidate"
      return
    fi
  done

  if command -v wslpath >/dev/null 2>&1 && [[ -n "${USERPROFILE:-}" ]]; then
    win_home="$(wslpath "$USERPROFILE")"
    for candidate in \
      "${win_home}/.codex/skills/skill-home-manager" \
      "${win_home}/.agents/skills/skill-home-manager"
    do
      if is_skill_home_manager_root "$candidate"; then
        printf '%s\n' "$candidate"
        return
      fi
    done
  fi

  for candidate in \
    "/mnt/c/Users/${USER}/.codex/skills/skill-home-manager" \
    "/mnt/c/Users/${USER}/.agents/skills/skill-home-manager"
  do
    if is_skill_home_manager_root "$candidate"; then
      printf '%s\n' "$candidate"
      return
    fi
  done

  if [[ "${SKILL_HOME_MANAGER_TEST_DISABLE_SCRIPT_ROOT:-0}" != "1" ]]; then
    script_dir="$(skill_home_manager_script_dir)"
    script_root="$(cd -- "${script_dir}/.." && pwd)"
    if is_skill_home_manager_root "$script_root"; then
      printf '%s\n' "$script_root"
      return
    fi
  fi

  echo "未找到已安装的 skill-home-manager 根目录，请先检查 \$CODEX_HOME/skills、~/.codex/skills 或 ~/.agents/skills" >&2
  return 1
}
