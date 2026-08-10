#!/usr/bin/env bash
set -euo pipefail

script_dir="$(
  cd -- "$(dirname -- "${BASH_SOURCE[0]}")"
  pwd
)"

server=""
no_browser=0
oauth_timeout=""

usage() {
  cat <<'EOF'
Usage: ensure-oauth-login.sh [--server <url>] [--no-browser] [--oauth-timeout <duration>]

Ensure skill-home has a valid registry login. Reuse an existing credential when
possible; otherwise open the browser OAuth flow, wait for approval, save the CLI
credential, and verify the resulting identity.

Flags:
  --server <url>             Authorize against a specific registry endpoint
  --no-browser               Print the authorization URL without opening it
  --oauth-timeout <duration> Maximum OAuth wait, for example 15m
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --server)
      if [[ $# -lt 2 || -z "$2" ]]; then
        usage >&2
        exit 1
      fi
      server="$2"
      shift 2
      ;;
    --server=*)
      server="${1#*=}"
      shift
      ;;
    --no-browser)
      no_browser=1
      shift
      ;;
    --oauth-timeout)
      if [[ $# -lt 2 || -z "$2" ]]; then
        usage >&2
        exit 1
      fi
      oauth_timeout="$2"
      shift 2
      ;;
    --oauth-timeout=*)
      oauth_timeout="${1#*=}"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

resolve_cli() {
  local published_target="${SKILL_HOME_INSTALL_DIR:-${HOME}/.local/bin}/skill-home"

  if command -v skill-home >/dev/null 2>&1; then
    command -v skill-home
    return 0
  fi
  if [[ -x "$published_target" ]]; then
    printf '%s\n' "$published_target"
    return 0
  fi
  return 1
}

supports_oauth() {
  local cli_path="$1"
  "$cli_path" login --help 2>&1 | grep -q -- '--no-browser'
}

cli="$(resolve_cli || true)"
if [[ -z "$cli" ]]; then
  bash "$script_dir/bootstrap-cli.sh"
  hash -r
  cli="$(resolve_cli || true)"
fi

if [[ -z "$cli" ]]; then
  echo "skill-home installation completed but the executable could not be located." >&2
  exit 2
fi

# With no endpoint override, preserve any valid config or environment credential.
# Keep identity output for the agent, but never print the credential itself.
if [[ -z "$server" ]]; then
  if identity="$("$cli" whoami 2>/dev/null)"; then
    printf '%s\n' "$identity"
    exit 0
  fi
fi

# An invalid environment key would override a valid OAuth credential saved to
# config. Remove it only inside this helper, then retry the persisted login.
if [[ -n "${SKILL_HOME_API_KEY:-}" ]]; then
  echo "The existing SKILL_HOME_API_KEY is not usable for this login; ignoring it inside this helper without exposing it."
  unset SKILL_HOME_API_KEY
  if [[ -z "$server" ]] && identity="$("$cli" whoami 2>/dev/null)"; then
    printf '%s\n' "$identity"
    exit 0
  fi
fi

if ! supports_oauth "$cli"; then
  echo "The installed skill-home CLI does not support OAuth; refreshing the published CLI..."
  bash "$script_dir/bootstrap-cli.sh" --force-reinstall
  hash -r

  published_target="${SKILL_HOME_INSTALL_DIR:-${HOME}/.local/bin}/skill-home"
  if [[ -x "$published_target" ]]; then
    cli="$published_target"
  else
    cli="$(resolve_cli || true)"
  fi
fi

if [[ -z "$cli" ]] || ! supports_oauth "$cli"; then
  echo "The latest published skill-home CLI still lacks OAuth support. Publish or install an OAuth-capable CLI before retrying; do not ask the user to create an API Key manually." >&2
  exit 3
fi

login_args=(login)
if [[ -n "$server" ]]; then
  login_args+=(--server "$server")
fi
if [[ "$no_browser" -eq 1 ]]; then
  login_args+=(--no-browser)
fi
if [[ -n "$oauth_timeout" ]]; then
  login_args+=(--oauth-timeout "$oauth_timeout")
fi

"$cli" "${login_args[@]}"
"$cli" whoami
