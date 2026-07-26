#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_SCRIPT="$(cd "${SCRIPT_DIR}/../.." && pwd)/deploy-update.sh"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

DEPLOY_DIR="${TMP_DIR}/deploy"
FAKE_BIN_DIR="${TMP_DIR}/bin"
mkdir -p "${DEPLOY_DIR}" "${FAKE_BIN_DIR}"

printf 'old installer\n' > "${DEPLOY_DIR}/install.ps1"
printf 'new installer\n' > "${DEPLOY_DIR}/install.ps1.new"

cat > "${FAKE_BIN_DIR}/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '<!doctype html><title>Skill Home</title>\n'
EOF

cat > "${FAKE_BIN_DIR}/systemctl" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF

cat > "${FAKE_BIN_DIR}/sleep" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF

chmod +x "${FAKE_BIN_DIR}/curl" "${FAKE_BIN_DIR}/systemctl" "${FAKE_BIN_DIR}/sleep"

set +e
output="$(
  PATH="${FAKE_BIN_DIR}:${PATH}" \
  DEPLOY_DIR="${DEPLOY_DIR}" \
  SKILL_HOME_WINDOWS_INSTALLCHECK_URL="https://example.test/skill-home/install.ps1" \
  bash "${TARGET_SCRIPT}" 2>&1
)"
status=$?
set -e

if [ "$status" -eq 0 ]; then
  echo "deploy-update should fail when the Windows installer route returns the Web page" >&2
  exit 41
fi

if ! grep -F "健康检查失败，正在回滚" <<<"$output" >/dev/null; then
  echo "deploy-update did not report rollback after verification failure" >&2
  exit 42
fi

if [ "$(cat "${DEPLOY_DIR}/install.ps1")" != "old installer" ]; then
  echo "deploy-update did not restore the previous Windows installer" >&2
  exit 43
fi

if [ -e "${DEPLOY_DIR}/install.ps1.old" ]; then
  echo "deploy-update left the rollback staging file behind" >&2
  exit 44
fi

echo "test-deploy-update: ok"
