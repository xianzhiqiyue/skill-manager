#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
TARGET_SCRIPT="${SCRIPT_DIR}/common.sh"

if [[ ! -f "${TARGET_SCRIPT}" ]]; then
  echo "missing common.sh: ${TARGET_SCRIPT}" >&2
  exit 1
fi

assert_eq() {
  local actual="$1"
  local expected="$2"
  local message="$3"

  if [[ "${actual}" != "${expected}" ]]; then
    echo "${message}: expected '${expected}', got '${actual}'" >&2
    exit 1
  fi
}

run_with_env() {
  local home_dir="$1"
  local expected_root="$2"
  local expected_skills_dir="$3"
  local expected_xigua_skills_dir="$4"
  shift 4

  (
    export HOME="${home_dir}"
    export USER="tester"
    unset CODEX_HOME
    unset SKILL_HOME_MANAGER_ROOT
    unset SKILL_HOME_CODEX_SKILLS_DIR
    unset SKILL_HOME_XIGUA_SKILLS_DIR
    unset SKILL_HOME_CONFIG
    unset USERPROFILE
    export SKILL_HOME_MANAGER_TEST_DISABLE_SCRIPT_ROOT=1
    for assignment in "$@"; do
      export "${assignment}"
    done

    # shellcheck source=/dev/null
    source "${TARGET_SCRIPT}"
    local resolved_root
    local resolved_skills_dir
    local resolved_xigua_skills_dir
    resolved_root="$(resolve_skill_home_manager_root)"
    resolved_skills_dir="$(resolve_codex_skills_dir)"
    resolved_xigua_skills_dir="$(resolve_xigua_skills_dir)"
    assert_eq "${resolved_root}" "${expected_root}" "resolve_skill_home_manager_root"
    assert_eq "${resolved_skills_dir}" "${expected_skills_dir}" "resolve_codex_skills_dir"
    assert_eq "${resolved_xigua_skills_dir}" "${expected_xigua_skills_dir}" "resolve_xigua_skills_dir"
  )
}

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

CONFIG_HOME="${TMP_DIR}/config-home"
mkdir -p "${CONFIG_HOME}/.config/skill-home" "${CONFIG_HOME}/custom-codex/skill-home-manager"
cat > "${CONFIG_HOME}/custom-codex/skill-home-manager/SKILL.md" <<'EOF'
---
name: skill-home-manager
---
EOF
cat > "${CONFIG_HOME}/.config/skill-home/config.yaml" <<'EOF'
ide:
  codex:
    global_path: ~/custom-codex
  xigua:
    global_path: ~/custom-xigua
EOF

run_with_env "${CONFIG_HOME}" "${CONFIG_HOME}/custom-codex/skill-home-manager" "${CONFIG_HOME}/custom-codex" "${CONFIG_HOME}/custom-xigua"

CODEX_HOME_ROOT="${TMP_DIR}/codex-home"
mkdir -p "${CODEX_HOME_ROOT}/skills/skill-home-manager"
cat > "${CODEX_HOME_ROOT}/skills/skill-home-manager/SKILL.md" <<'EOF'
---
name: skill-home-manager
---
EOF
run_with_env "${TMP_DIR}/home-two" "${CODEX_HOME_ROOT}/skills/skill-home-manager" "${CODEX_HOME_ROOT}/skills" "${TMP_DIR}/home-two/.xigua-agent/skills" \
  "CODEX_HOME=${CODEX_HOME_ROOT}"

AGENTS_HOME="${TMP_DIR}/home-agents"
mkdir -p "${AGENTS_HOME}/.agents/skills/skill-home-manager"
cat > "${AGENTS_HOME}/.agents/skills/skill-home-manager/SKILL.md" <<'EOF'
---
name: skill-home-manager
---
EOF
run_with_env "${AGENTS_HOME}" "${AGENTS_HOME}/.agents/skills/skill-home-manager" "${AGENTS_HOME}/.agents/skills" "${AGENTS_HOME}/.xigua-agent/skills"

OVERRIDE_ROOT="${TMP_DIR}/override-root"
mkdir -p "${OVERRIDE_ROOT}"
cat > "${OVERRIDE_ROOT}/SKILL.md" <<'EOF'
---
name: skill-home-manager
---
EOF
run_with_env "${TMP_DIR}/home-three" "${OVERRIDE_ROOT}" "${TMP_DIR}/home-three/.agents/skills" "${TMP_DIR}/home-three/.xigua-agent/skills" \
  "SKILL_HOME_MANAGER_ROOT=${OVERRIDE_ROOT}"

WINDOWS_HOME="${TMP_DIR}/windows-home"
mkdir -p "${WINDOWS_HOME}/.codex/skills/skill-home-manager"
cat > "${WINDOWS_HOME}/.codex/skills/skill-home-manager/SKILL.md" <<'EOF'
---
name: skill-home-manager
---
EOF

MOCK_WSL_BIN="${TMP_DIR}/mock-wsl-bin"
mkdir -p "${MOCK_WSL_BIN}"
cat > "${MOCK_WSL_BIN}/wslpath" <<EOF
#!/usr/bin/env bash
printf '%s\n' "${WINDOWS_HOME}"
EOF
chmod +x "${MOCK_WSL_BIN}/wslpath"

cat > "${MOCK_WSL_BIN}/cmd.exe" <<'EOF'
#!/usr/bin/env bash
printf 'C:\Users\WinTester\r\n'
EOF
chmod +x "${MOCK_WSL_BIN}/cmd.exe"

run_with_env "${TMP_DIR}/home-wsl" "${WINDOWS_HOME}/.codex/skills/skill-home-manager" "${WINDOWS_HOME}/.codex/skills" "${TMP_DIR}/home-wsl/.xigua-agent/skills" \
  "PATH=${MOCK_WSL_BIN}:${PATH}"

LEGACY_HOME="${TMP_DIR}/home-legacy"
mkdir -p "${LEGACY_HOME}/.codex/skills/skill-home-manager"
cat > "${LEGACY_HOME}/.codex/skills/skill-home-manager/SKILL.md" <<'EOF'
---
name: skill-home-manager
---
EOF
run_with_env "${LEGACY_HOME}" "${LEGACY_HOME}/.codex/skills/skill-home-manager" "${LEGACY_HOME}/.codex/skills" "${LEGACY_HOME}/.xigua-agent/skills"

XIGUA_ONLY_HOME="${TMP_DIR}/home-xigua-only"
mkdir -p "${XIGUA_ONLY_HOME}/.xigua-agent/skills/skill-home-manager"
cat > "${XIGUA_ONLY_HOME}/.xigua-agent/skills/skill-home-manager/SKILL.md" <<'EOF'
---
name: skill-home-manager
---
EOF
run_with_env "${XIGUA_ONLY_HOME}" "${XIGUA_ONLY_HOME}/.xigua-agent/skills/skill-home-manager" "${XIGUA_ONLY_HOME}/.agents/skills" "${XIGUA_ONLY_HOME}/.xigua-agent/skills"

CONTEXT_SKILL_DIR="${TMP_DIR}/context-home/.xigua-agent/skills/skill-home-manager"
mkdir -p "${CONTEXT_SKILL_DIR}"
cat > "${CONTEXT_SKILL_DIR}/SKILL.md" <<'EOF'
---
name: skill-home-manager
---
EOF

(
  export HOME="${TMP_DIR}/context-home"
  # shellcheck source=/dev/null
  source "${TARGET_SCRIPT}"
  context_file="$(write_platform_context auto "${CONTEXT_SKILL_DIR}" global "${TMP_DIR}/context-home/.xigua-agent/skills" mirror)"
  if [[ ! -f "${context_file}" ]]; then
    echo "platform context file was not created: ${context_file}" >&2
    exit 1
  fi
  grep -q '"platform": "xigua"' "${context_file}"
  grep -q '"skill_name": "skill-home-manager"' "${context_file}"
  grep -q '"install_mode": "mirror"' "${context_file}"
)

echo "test-common: ok"
