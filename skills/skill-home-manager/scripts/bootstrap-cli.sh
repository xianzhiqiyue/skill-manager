#!/usr/bin/env bash
set -euo pipefail

primary_install_url="${SKILL_HOME_INSTALL_URL:-https://soulstore.ciqtek.com/skill-home/install.sh}"
fallback_install_url="${SKILL_HOME_GITHUB_INSTALL_URL:-https://raw.githubusercontent.com/xianzhiqiyue/skill-manager/main/skill-home-cli/install.sh}"
user_bin_dir="${HOME}/.local/bin"
install_dir="${SKILL_HOME_INSTALL_DIR:-$user_bin_dir}"
user_install_target="$install_dir/skill-home"
system_bin_dir="/usr/local/bin"
system_install_target="$system_bin_dir/skill-home"
force_reinstall=0
system_install=0
version=""

usage() {
  cat <<'EOF'
Usage: bootstrap-cli.sh [--force-reinstall] [--version <tag>] [--system]

Ensure the published skill-home CLI is available. This script downloads the
public installer from the deployed site, and falls back to GitHub if needed.

Flags:
  --force-reinstall Reinstall even if `skill-home` already exists in PATH
  --force-rebuild   Backward-compatible alias of --force-reinstall
  --version <tag>   Install a specific release, e.g. v0.2.4
  --system          Install to /usr/local/bin/skill-home instead of ~/.local/bin
EOF
}

download_script() {
  local url="$1"
  local output="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "$output" "$url"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -qO "$output" "$url"
    return
  fi

  echo "需要 curl 或 wget 来下载安装脚本" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --force-reinstall|--force-rebuild)
      force_reinstall=1
      shift
      ;;
    --version)
      if [[ $# -lt 2 ]]; then
        usage
        exit 1
      fi
      version="$2"
      force_reinstall=1
      shift 2
      ;;
    --version=*)
      version="${1#*=}"
      force_reinstall=1
      shift
      ;;
    --system)
      system_install=1
      force_reinstall=1
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

if [[ "$force_reinstall" -eq 0 ]] && command -v skill-home >/dev/null 2>&1; then
  existing_cli="$(command -v skill-home)"
  echo "skill-home already available in PATH: $existing_cli"
  "$existing_cli" version 2>/dev/null || "$existing_cli" --version 2>/dev/null || true
  exit 0
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
installer="$tmp_dir/install.sh"

if download_script "$primary_install_url" "$installer"; then
  echo "Using installer from $primary_install_url"
else
  echo "Primary installer unavailable, falling back to $fallback_install_url"
  download_script "$fallback_install_url" "$installer"
fi

chmod +x "$installer"

if [[ "$system_install" -eq 1 ]]; then
  if [[ -n "$version" ]]; then
    sudo env SKILL_HOME_INSTALL_DIR="$system_bin_dir" bash "$installer" "$version"
  else
    sudo env SKILL_HOME_INSTALL_DIR="$system_bin_dir" bash "$installer"
  fi
  "$system_install_target" version 2>/dev/null || "$system_install_target" --version 2>/dev/null || true
  exit 0
fi

if [[ -n "$version" ]]; then
  env SKILL_HOME_INSTALL_DIR="$install_dir" bash "$installer" "$version"
else
  env SKILL_HOME_INSTALL_DIR="$install_dir" bash "$installer"
fi

if command -v skill-home >/dev/null 2>&1; then
  hash -r
  skill-home version 2>/dev/null || skill-home --version 2>/dev/null || true
  exit 0
fi

"$user_install_target" version 2>/dev/null || "$user_install_target" --version 2>/dev/null || true
