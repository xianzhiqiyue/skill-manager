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

json_escape() {
  local value="${1:-}"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '%s' "$value"
}

parse_skill_name_from_file() {
  local skill_file="$1"

  if [[ ! -f "$skill_file" ]]; then
    return 1
  fi

  sed -n 's/^name:[[:space:]]*//p' "$skill_file" \
    | head -n 1 \
    | tr -d '"' \
    | tr -d "'"
}

detect_platform_from_skill_dir() {
  local skill_dir="${1:-}"

  case "$skill_dir" in
    *"/.xigua-agent/skills/"*|*"/.xigua/skills/"*)
      printf '%s\n' "xigua"
      return
      ;;
    *"/.agents/skills/"*|*"/.codex/skills/"*)
      printf '%s\n' "codex"
      return
      ;;
    *"/.claude/skills/"*)
      printf '%s\n' "claude"
      return
      ;;
  esac

  if [[ -f "${skill_dir%/}/skill.json" ]]; then
    printf '%s\n' "xigua"
    return
  fi

  printf '%s\n' "unknown"
}

write_platform_context() {
  local platform="${1:-auto}"
  local skill_dir="${2:-}"
  local target_scope="${3:-unknown}"
  local target_path="${4:-}"
  local install_mode="${5:-unknown}"
  local skill_name=""
  local context_dir=""
  local context_file=""
  local tmp_file=""
  local updated_at=""

  if [[ -z "$skill_dir" || ! -d "$skill_dir" ]]; then
    echo "Invalid installed skill directory: $skill_dir" >&2
    return 1
  fi

  skill_dir="$(
    cd -- "$skill_dir"
    pwd
  )"

  if [[ "$platform" == "auto" || -z "$platform" ]]; then
    platform="$(detect_platform_from_skill_dir "$skill_dir")"
  fi

  if [[ -z "$target_path" ]]; then
    target_path="$(dirname "$skill_dir")"
  fi

  skill_name="$(parse_skill_name_from_file "${skill_dir%/}/SKILL.md" || true)"
  if [[ -z "$skill_name" ]]; then
    skill_name="$(basename "$skill_dir")"
  fi

  context_dir="${skill_dir%/}/.skill-home"
  context_file="${context_dir}/platform-context.json"
  tmp_file="${context_file}.tmp"
  updated_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  mkdir -p "$context_dir"
  cat > "$tmp_file" <<EOF
{
  "schema_version": 1,
  "skill_name": "$(json_escape "$skill_name")",
  "platform": "$(json_escape "$platform")",
  "installed_skill_dir": "$(json_escape "$skill_dir")",
  "target_scope": "$(json_escape "$target_scope")",
  "target_path": "$(json_escape "$target_path")",
  "install_mode": "$(json_escape "$install_mode")",
  "source": "skill-home-manager",
  "updated_at": "$(json_escape "$updated_at")"
}
EOF
  mv "$tmp_file" "$context_file"
  printf '%s\n' "$context_file"
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

parse_xigua_global_path_from_config() {
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
      in_xigua=(line=="xigua")
      next
    }
    in_xigua && /^[[:space:]]*global_path:[[:space:]]*/ {
      sub(/^[[:space:]]*global_path:[[:space:]]*/, "", $0)
      gsub(/["'\'']/, "", $0)
      print $0
      exit
    }
  ' "$config_path"
}

resolve_windows_home() {
  local raw_home=""
  local win_home=""

  if ! command -v wslpath >/dev/null 2>&1; then
    return 1
  fi

  if [[ -n "${USERPROFILE:-}" ]]; then
    win_home="$(wslpath "$USERPROFILE" 2>/dev/null || true)"
    if [[ -n "$win_home" ]]; then
      printf '%s\n' "$win_home"
      return 0
    fi
  fi

  if command -v cmd.exe >/dev/null 2>&1; then
    raw_home="$(
      cmd.exe /c "echo %USERPROFILE%" 2>/dev/null \
        | tr -d '\r' \
        | awk 'NF { value=$0 } END { print value }'
    )"
    if [[ -n "$raw_home" && "$raw_home" != "%USERPROFILE%" ]]; then
      win_home="$(wslpath "$raw_home" 2>/dev/null || true)"
      if [[ -n "$win_home" ]]; then
        printf '%s\n' "$win_home"
        return 0
      fi
    fi
  fi

  return 1
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

  if [[ -d "${HOME}/.agents" ]]; then
    printf '%s\n' "${HOME}/.agents/skills"
    return
  fi

  if [[ -d "${HOME}/.codex" ]]; then
    printf '%s\n' "${HOME}/.codex/skills"
    return
  fi

  win_home="$(resolve_windows_home || true)"
  if [[ -n "$win_home" ]]; then
    if [[ -d "${win_home}/.agents" ]]; then
      printf '%s\n' "${win_home}/.agents/skills"
      return
    fi
    if [[ -d "${win_home}/.codex" ]]; then
      printf '%s\n' "${win_home}/.codex/skills"
      return
    fi
  fi

  if [[ -d "/mnt/c/Users/${USER}/.agents" ]]; then
    printf '%s\n' "/mnt/c/Users/${USER}/.agents/skills"
    return
  fi

  if [[ -d "/mnt/c/Users/${USER}/.codex" ]]; then
    printf '%s\n' "/mnt/c/Users/${USER}/.codex/skills"
    return
  fi

  printf '%s\n' "${HOME}/.agents/skills"
}

resolve_xigua_skills_dir() {
  local parsed=""
  local config_path=""

  if [[ -n "${SKILL_HOME_XIGUA_SKILLS_DIR:-}" ]]; then
    expand_path "$SKILL_HOME_XIGUA_SKILLS_DIR"
    return
  fi

  config_path="${SKILL_HOME_CONFIG:-${HOME}/.config/skill-home/config.yaml}"
  parsed="$(parse_xigua_global_path_from_config "$config_path" || true)"
  if [[ -n "$parsed" ]]; then
    expand_path "$parsed"
    return
  fi

  printf '%s\n' "${HOME}/.xigua-agent/skills"
}

list_codex_skills_dir_candidates() {
  local config_path=""
  local parsed=""
  local win_home=""
  local candidate=""
  declare -A seen=()

  config_path="${SKILL_HOME_CONFIG:-${HOME}/.config/skill-home/config.yaml}"

  for candidate in \
    "${SKILL_HOME_CODEX_SKILLS_DIR:-}" \
    "$(parse_codex_global_path_from_config "$config_path" || true)" \
    "${CODEX_HOME:+$(expand_path "$CODEX_HOME")/skills}" \
    "${HOME}/.agents/skills" \
    "${HOME}/.codex/skills"
  do
    candidate="$(expand_path "$candidate")"
    if [[ -n "$candidate" && -z "${seen[$candidate]:-}" ]]; then
      seen["$candidate"]=1
      printf '%s\n' "$candidate"
    fi
  done

  win_home="$(resolve_windows_home || true)"
  if [[ -n "$win_home" ]]; then
    for candidate in \
      "${win_home}/.agents/skills" \
      "${win_home}/.codex/skills"
    do
      if [[ -z "${seen[$candidate]:-}" ]]; then
        seen["$candidate"]=1
        printf '%s\n' "$candidate"
      fi
    done
  fi

  for candidate in \
    "/mnt/c/Users/${USER}/.agents/skills" \
    "/mnt/c/Users/${USER}/.codex/skills"
  do
    if [[ -z "${seen[$candidate]:-}" ]]; then
      seen["$candidate"]=1
      printf '%s\n' "$candidate"
    fi
  done
}

list_xigua_skills_dir_candidates() {
  local config_path=""
  local candidate=""
  declare -A seen=()

  config_path="${SKILL_HOME_CONFIG:-${HOME}/.config/skill-home/config.yaml}"

  for candidate in \
    "${SKILL_HOME_XIGUA_SKILLS_DIR:-}" \
    "$(parse_xigua_global_path_from_config "$config_path" || true)" \
    "${HOME}/.xigua-agent/skills"
  do
    candidate="$(expand_path "$candidate")"
    if [[ -n "$candidate" && -z "${seen[$candidate]:-}" ]]; then
      seen["$candidate"]=1
      printf '%s\n' "$candidate"
    fi
  done
}

resolve_installed_codex_skill_dir() {
  local skill_name="$1"
  local candidate=""

  while IFS= read -r candidate; do
    if [[ -f "${candidate%/}/${skill_name}/SKILL.md" ]]; then
      printf '%s\n' "${candidate%/}/${skill_name}"
      return 0
    fi
  done < <(list_codex_skills_dir_candidates)

  return 1
}

resolve_installed_xigua_skill_dir() {
  local skill_name="$1"
  local candidate=""

  while IFS= read -r candidate; do
    if [[ -f "${candidate%/}/${skill_name}/SKILL.md" ]]; then
      printf '%s\n' "${candidate%/}/${skill_name}"
      return 0
    fi
  done < <(list_xigua_skills_dir_candidates)

  return 1
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
  local configured_xigua_skills_dir=""
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

  configured_xigua_skills_dir="$(resolve_xigua_skills_dir)"
  candidate="${configured_xigua_skills_dir%/}/skill-home-manager"
  if is_skill_home_manager_root "$candidate"; then
    printf '%s\n' "$candidate"
    return
  fi

  for candidate in \
    "${HOME}/.agents/skills/skill-home-manager" \
    "${HOME}/.codex/skills/skill-home-manager" \
    "${HOME}/.xigua-agent/skills/skill-home-manager"
  do
    if is_skill_home_manager_root "$candidate"; then
      printf '%s\n' "$candidate"
      return
    fi
  done

  win_home="$(resolve_windows_home || true)"
  if [[ -n "$win_home" ]]; then
    for candidate in \
      "${win_home}/.agents/skills/skill-home-manager" \
      "${win_home}/.codex/skills/skill-home-manager"
    do
      if is_skill_home_manager_root "$candidate"; then
        printf '%s\n' "$candidate"
        return
      fi
    done
  fi

  for candidate in \
    "/mnt/c/Users/${USER}/.agents/skills/skill-home-manager" \
    "/mnt/c/Users/${USER}/.codex/skills/skill-home-manager"
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

  echo "未找到已安装的 skill-home-manager 根目录，请先检查 skill-home 配置、\$CODEX_HOME/skills、~/.agents/skills 或 ~/.codex/skills" >&2
  return 1
}
