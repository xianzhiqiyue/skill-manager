#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: promote-hosted-release.sh <version> <dist-dir> <install-script>

将 GitHub Actions 中已打包好的 CLI 发布产物同步到 Skill Home 服务端，并触发原子切换。

环境变量:
  SKILL_HOME_DEPLOY_HOST        目标主机，例如 47.122.112.210
  SKILL_HOME_DEPLOY_USER        目标用户，默认 root
  SKILL_HOME_DEPLOY_PORT        SSH 端口，默认 22
  SKILL_HOME_DEPLOY_DIR         部署目录，默认 /opt/skill-home
  SKILL_HOME_PUBLIC_BASE_URL    对外验收地址，默认 http://<host>:8080
EOF
}

require_file() {
  local path="$1"
  if [ ! -f "$path" ]; then
    echo "缺少文件: $path" >&2
    exit 1
  fi
}

VERSION="${1:-}"
DIST_DIR="${2:-}"
INSTALL_SCRIPT_PATH="${3:-}"

if [ -z "$VERSION" ] || [ -z "$DIST_DIR" ] || [ -z "$INSTALL_SCRIPT_PATH" ]; then
  usage
  exit 1
fi

if [[ "$VERSION" != v* ]]; then
  VERSION="v${VERSION}"
fi

DEPLOY_HOST="${SKILL_HOME_DEPLOY_HOST:-}"
DEPLOY_USER="${SKILL_HOME_DEPLOY_USER:-root}"
DEPLOY_PORT="${SKILL_HOME_DEPLOY_PORT:-22}"
DEPLOY_DIR="${SKILL_HOME_DEPLOY_DIR:-/opt/skill-home}"
PUBLIC_BASE_URL="${SKILL_HOME_PUBLIC_BASE_URL:-}"

if [ -z "$DEPLOY_HOST" ]; then
  echo "缺少环境变量: SKILL_HOME_DEPLOY_HOST" >&2
  exit 1
fi

if [ -z "$PUBLIC_BASE_URL" ]; then
  PUBLIC_BASE_URL="http://${DEPLOY_HOST}:8080"
fi

for cmd in ssh scp curl python3; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "缺少命令: $cmd" >&2
    exit 1
  fi
done

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIST_ABS="$(cd "$DIST_DIR" && pwd)"
INSTALL_SCRIPT_ABS="$(cd "$(dirname "$INSTALL_SCRIPT_PATH")" && pwd)/$(basename "$INSTALL_SCRIPT_PATH")"
DEPLOY_SCRIPT_ABS="${ROOT_DIR}/deploy-update.sh"

require_file "$INSTALL_SCRIPT_ABS"
require_file "$DEPLOY_SCRIPT_ABS"

required_assets=(
  checksums.txt
  latest.json
  skill-home-darwin-amd64.tar.gz
  skill-home-darwin-arm64.tar.gz
  skill-home-linux-amd64.tar.gz
  skill-home-linux-arm64.tar.gz
  skill-home-windows-amd64.zip
)

for asset in "${required_assets[@]}"; do
  require_file "${DIST_ABS}/${asset}"
done

python3 - "$DIST_ABS/latest.json" "$VERSION" <<'PY'
import json
import pathlib
import sys

latest_path = pathlib.Path(sys.argv[1])
expected = sys.argv[2]
payload = json.loads(latest_path.read_text())
actual = payload.get("tag_name", "").strip()
if actual != expected:
    raise SystemExit(f"latest.json 中的 tag_name 为 {actual!r}，期望 {expected!r}")
PY

ssh_opts=(-p "$DEPLOY_PORT")
remote_target="${DEPLOY_USER}@${DEPLOY_HOST}"
remote_tmp="${DEPLOY_DIR}/.cli-release-${VERSION}-$(date +%s)"

echo "正在准备远程暂存目录..."
ssh "${ssh_opts[@]}" "$remote_target" "rm -rf '$remote_tmp' '$DEPLOY_DIR/releases.new' && mkdir -p '$remote_tmp'"

echo "正在上传 CLI 发布产物..."
scp "${ssh_opts[@]}" \
  "${DIST_ABS}/checksums.txt" \
  "${DIST_ABS}/latest.json" \
  "${DIST_ABS}/skill-home-darwin-amd64.tar.gz" \
  "${DIST_ABS}/skill-home-darwin-arm64.tar.gz" \
  "${DIST_ABS}/skill-home-linux-amd64.tar.gz" \
  "${DIST_ABS}/skill-home-linux-arm64.tar.gz" \
  "${DIST_ABS}/skill-home-windows-amd64.zip" \
  "${INSTALL_SCRIPT_ABS}" \
  "${DEPLOY_SCRIPT_ABS}" \
  "${remote_target}:${remote_tmp}/"

echo "正在切换 Skill Home hosted release..."
ssh "${ssh_opts[@]}" "$remote_target" "bash -s" -- "$DEPLOY_DIR" "$remote_tmp" "$VERSION" <<'EOF'
set -euo pipefail

deploy_dir="$1"
remote_tmp="$2"
version="$3"
stage_dir="${deploy_dir}/releases.new"

rm -rf "$stage_dir"
mkdir -p "$stage_dir"
if [ -d "${deploy_dir}/releases" ]; then
  cp -a "${deploy_dir}/releases/." "$stage_dir/"
fi

rm -rf "${stage_dir}/${version}"
mkdir -p "${stage_dir}/${version}"

cp "${remote_tmp}/latest.json" "${stage_dir}/latest.json"
for asset in \
  checksums.txt \
  skill-home-darwin-amd64.tar.gz \
  skill-home-darwin-arm64.tar.gz \
  skill-home-linux-amd64.tar.gz \
  skill-home-linux-arm64.tar.gz \
  skill-home-windows-amd64.zip
do
  cp "${remote_tmp}/${asset}" "${stage_dir}/${version}/${asset}"
done

install -m 0755 "${remote_tmp}/install.sh" "${deploy_dir}/install.sh.new"
DEPLOY_DIR="$deploy_dir" bash "${remote_tmp}/deploy-update.sh"
rm -rf "$remote_tmp"
EOF

echo "正在验收 Skill Home hosted release..."
curl -fsSL "${PUBLIC_BASE_URL%/}/install.sh" | grep -F "${PUBLIC_BASE_URL%/}/releases" >/dev/null
curl -fsSL "${PUBLIC_BASE_URL%/}/releases/latest.json" | grep -F "\"tag_name\":\"${VERSION}\"" >/dev/null
curl -fsSI "${PUBLIC_BASE_URL%/}/releases/${VERSION}/checksums.txt" >/dev/null
curl -fsSI "${PUBLIC_BASE_URL%/}/releases/${VERSION}/skill-home-linux-amd64.tar.gz" >/dev/null

echo "✅ Skill Home hosted release 已同步到 ${PUBLIC_BASE_URL%/}"
