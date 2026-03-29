#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_DIR="$(cd "${ROOT_DIR}/.." && pwd)"
TARGET_SCRIPT="${SCRIPT_DIR}/promote-hosted-release.sh"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

DIST_DIR="${TMP_DIR}/dist"
FAKE_BIN_DIR="${TMP_DIR}/bin"
INSTALL_SCRIPT_PATH="${TMP_DIR}/install.sh"
SCP_ARGS_PATH="${TMP_DIR}/scp.args"
SSH_ARGS_PATH="${TMP_DIR}/ssh.args"

mkdir -p "${DIST_DIR}" "${FAKE_BIN_DIR}"

cat > "${DIST_DIR}/latest.json" <<'EOF'
{"tag_name":"v0.0.0"}
EOF

for asset in \
  checksums.txt \
  skill-home-darwin-amd64.tar.gz \
  skill-home-darwin-arm64.tar.gz \
  skill-home-linux-amd64.tar.gz \
  skill-home-linux-arm64.tar.gz \
  skill-home-windows-amd64.zip
do
  printf 'fixture\n' > "${DIST_DIR}/${asset}"
done

cat > "${INSTALL_SCRIPT_PATH}" <<'EOF'
#!/usr/bin/env bash
echo install
EOF

cat > "${FAKE_BIN_DIR}/ssh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "$@" >> "__SSH_ARGS_PATH__"

if [[ " $* " == *" bash -s "* ]]; then
  cat >/dev/null
fi
exit 0
EOF

sed -i "s|__SSH_ARGS_PATH__|${SSH_ARGS_PATH}|g" "${FAKE_BIN_DIR}/ssh"

cat > "${FAKE_BIN_DIR}/scp" <<EOF
#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "\$@" > "${SCP_ARGS_PATH}"
if [[ "\${1:-}" != "-P" ]]; then
  echo "expected scp to receive -P, got: \${1:-<missing>}" >&2
  exit 42
fi
if [[ "\${2:-}" != "22" ]]; then
  echo "expected scp to receive port 22, got: \${2:-<missing>}" >&2
  exit 43
fi
exit 0
EOF

cat > "${FAKE_BIN_DIR}/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

case "$*" in
  *"/install.sh"*)
    printf 'https://example.test/skill-home/releases\n'
    ;;
  *"/releases/latest.json"*)
    printf '{"tag_name":"v0.0.0"}\n'
    ;;
  *)
    :
    ;;
esac
EOF

chmod +x "${FAKE_BIN_DIR}/ssh" "${FAKE_BIN_DIR}/scp" "${FAKE_BIN_DIR}/curl"

PATH="${FAKE_BIN_DIR}:${PATH}" \
SKILL_HOME_DEPLOY_HOST="example.test" \
SKILL_HOME_DEPLOY_USER="root" \
SKILL_HOME_DEPLOY_PORT="22" \
SKILL_HOME_DEPLOY_DIR="/opt/skill-home" \
SKILL_HOME_PUBLIC_BASE_URL="https://example.test/skill-home" \
bash "${TARGET_SCRIPT}" v0.0.0 "${DIST_DIR}" "${INSTALL_SCRIPT_PATH}"

if ! grep -qx -- '-P' "${SCP_ARGS_PATH}"; then
  echo "scp arguments did not include -P" >&2
  exit 44
fi

if ! grep -qx -- '22' "${SCP_ARGS_PATH}"; then
  echo "scp arguments did not include port 22" >&2
  exit 45
fi

if ! grep -qx -- 'https://example.test/skill-home/install.sh' "${SSH_ARGS_PATH}"; then
  echo "ssh arguments did not include prefixed install check url" >&2
  exit 46
fi

if ! grep -qx -- 'https://example.test/skill-home/releases/latest.json' "${SSH_ARGS_PATH}"; then
  echo "ssh arguments did not include prefixed releases latest url" >&2
  exit 47
fi

echo "test-promote-hosted-release: ok"
