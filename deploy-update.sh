#!/bin/bash
# Skill-Home 更新部署脚本
# 在目标服务器上执行，支持二进制更新和 CLI 静态发布产物更新

set -euo pipefail

echo "=== Skill-Home 更新部署 ==="

DEPLOY_DIR="${DEPLOY_DIR:-/opt/skill-home}"
BACKUP_DIR="${DEPLOY_DIR}/backup-$(date +%Y%m%d-%H%M%S)"
LOCAL_BASE_URL="${SKILL_HOME_LOCAL_BASE_URL:-http://localhost:8080/skill-home}"
HEALTHCHECK_URL="${SKILL_HOME_HEALTHCHECK_URL:-${LOCAL_BASE_URL}/health}"
INSTALLCHECK_URL="${SKILL_HOME_INSTALLCHECK_URL:-${LOCAL_BASE_URL}/install.sh}"
WINDOWS_INSTALLCHECK_URL="${SKILL_HOME_WINDOWS_INSTALLCHECK_URL:-${LOCAL_BASE_URL}/install.ps1}"
RELEASES_LATEST_URL="${SKILL_HOME_RELEASES_LATEST_URL:-${LOCAL_BASE_URL}/releases/latest.json}"

update_server=0
update_cli_binary=0
update_install_script=0
update_windows_install_script=0
update_release_assets=0
needs_restart=0

[ -f "$DEPLOY_DIR/server.new" ] && update_server=1 && needs_restart=1
[ -f "$DEPLOY_DIR/skill-home.new" ] && update_cli_binary=1 && needs_restart=1
[ -f "$DEPLOY_DIR/install.sh.new" ] && update_install_script=1
[ -f "$DEPLOY_DIR/install.ps1.new" ] && update_windows_install_script=1
[ -d "$DEPLOY_DIR/releases.new" ] && update_release_assets=1

if [ "$update_server" -eq 0 ] \
  && [ "$update_cli_binary" -eq 0 ] \
  && [ "$update_install_script" -eq 0 ] \
  && [ "$update_windows_install_script" -eq 0 ] \
  && [ "$update_release_assets" -eq 0 ]; then
  echo "❌ 没有发现可部署的更新文件"
  exit 1
fi

backup_if_exists() {
  local source_path="$1"
  local backup_name="$2"

  [ -e "$source_path" ] || return 0
  mkdir -p "$BACKUP_DIR"
  cp -a "$source_path" "${BACKUP_DIR}/${backup_name}"
}

replace_file_if_present() {
  local new_path="$1"
  local current_path="$2"
  local old_path="${current_path}.old"
  local mode="${3:-}"

  [ -f "$new_path" ] || return 0

  rm -f "$old_path"
  if [ -e "$current_path" ]; then
    mv "$current_path" "$old_path"
  fi
  mv "$new_path" "$current_path"
  if [ -n "$mode" ]; then
    chmod "$mode" "$current_path"
  fi
}

replace_dir_if_present() {
  local new_path="$1"
  local current_path="$2"
  local old_path="${current_path}.old"

  [ -d "$new_path" ] || return 0

  rm -rf "$old_path"
  if [ -e "$current_path" ]; then
    mv "$current_path" "$old_path"
  fi
  mv "$new_path" "$current_path"
}

restore_file_if_needed() {
  local current_path="$1"
  local old_path="${current_path}.old"

  if [ ! -e "$old_path" ]; then
    rm -f "$current_path"
    return 0
  fi

  rm -f "$current_path"
  mv -f "$old_path" "$current_path" || true
}

restore_dir_if_needed() {
  local current_path="$1"
  local old_path="${current_path}.old"

  if [ ! -d "$old_path" ]; then
    rm -rf "$current_path"
    return 0
  fi

  rm -rf "$current_path"
  mv "$old_path" "$current_path" || true
}

verify_deployment() {
  if [ "$needs_restart" -eq 1 ]; then
    curl -fsSL "$HEALTHCHECK_URL" >/dev/null
  fi

  if [ "$update_install_script" -eq 1 ] || [ -f "$DEPLOY_DIR/install.sh" ]; then
    curl -fsSL "$INSTALLCHECK_URL" | grep -q "skill-home CLI 安装脚本"
  fi
  if [ "$update_windows_install_script" -eq 1 ] || [ -f "$DEPLOY_DIR/install.ps1" ]; then
    curl -fsSL "$WINDOWS_INSTALLCHECK_URL" | grep -q "skill-home CLI Windows 安装脚本"
  fi

  if [ "$update_release_assets" -eq 1 ] || [ -d "$DEPLOY_DIR/releases" ]; then
    curl -fsSL "$RELEASES_LATEST_URL" | grep -q '"tag_name"'
  fi
}

rollback() {
  set +e
  echo "❌ 健康检查失败，正在回滚..."

  if [ "$needs_restart" -eq 1 ]; then
    systemctl stop skill-home || true
  fi

  if [ "$update_server" -eq 1 ]; then
    restore_file_if_needed "$DEPLOY_DIR/server"
  fi
  if [ "$update_cli_binary" -eq 1 ]; then
    restore_file_if_needed "$DEPLOY_DIR/skill-home"
  fi
  if [ "$update_install_script" -eq 1 ]; then
    restore_file_if_needed "$DEPLOY_DIR/install.sh"
  fi
  if [ "$update_windows_install_script" -eq 1 ]; then
    restore_file_if_needed "$DEPLOY_DIR/install.ps1"
  fi
  if [ "$update_release_assets" -eq 1 ]; then
    restore_dir_if_needed "$DEPLOY_DIR/releases"
  fi

  if [ "$needs_restart" -eq 1 ]; then
    systemctl start skill-home || true
  fi

  echo "✅ 已回滚到之前的版本"
  exit 1
}

echo "[1/5] 备份现有版本..."
if [ "$update_server" -eq 1 ]; then
  backup_if_exists "$DEPLOY_DIR/server" "server"
fi
if [ "$update_cli_binary" -eq 1 ]; then
  backup_if_exists "$DEPLOY_DIR/skill-home" "skill-home"
fi
if [ "$update_install_script" -eq 1 ]; then
  backup_if_exists "$DEPLOY_DIR/install.sh" "install.sh"
fi
if [ "$update_windows_install_script" -eq 1 ]; then
  backup_if_exists "$DEPLOY_DIR/install.ps1" "install.ps1"
fi
if [ "$update_release_assets" -eq 1 ]; then
  backup_if_exists "$DEPLOY_DIR/releases" "releases"
fi

if [ "$needs_restart" -eq 1 ]; then
  echo "[2/5] 停止 API 服务..."
  systemctl stop skill-home || true
else
  echo "[2/5] 跳过 API 重启（仅更新 CLI 静态产物）..."
fi

echo "[3/5] 更新部署文件..."
if [ "$update_server" -eq 1 ]; then
  replace_file_if_present "$DEPLOY_DIR/server.new" "$DEPLOY_DIR/server" 0755
fi
if [ "$update_cli_binary" -eq 1 ]; then
  replace_file_if_present "$DEPLOY_DIR/skill-home.new" "$DEPLOY_DIR/skill-home" 0755
fi
if [ "$update_install_script" -eq 1 ]; then
  replace_file_if_present "$DEPLOY_DIR/install.sh.new" "$DEPLOY_DIR/install.sh" 0755
fi
if [ "$update_windows_install_script" -eq 1 ]; then
  replace_file_if_present "$DEPLOY_DIR/install.ps1.new" "$DEPLOY_DIR/install.ps1" 0644
fi
if [ "$update_release_assets" -eq 1 ]; then
  replace_dir_if_present "$DEPLOY_DIR/releases.new" "$DEPLOY_DIR/releases"
fi

if [ "$needs_restart" -eq 1 ]; then
  echo "[4/5] 启动 API 服务..."
  systemctl start skill-home
else
  echo "[4/5] 跳过 API 启动（服务保持运行）..."
fi

echo "[5/5] 验收发布结果..."
sleep 2
if ! verify_deployment; then
  rollback
fi

echo "✅ 部署成功！服务运行正常"
echo ""
echo "版本信息:"
if [ -x "$DEPLOY_DIR/server" ]; then
  "$DEPLOY_DIR/server" --version 2>/dev/null || echo "server: 已更新"
fi
if [ -x "$DEPLOY_DIR/skill-home" ]; then
  "$DEPLOY_DIR/skill-home" --version 2>/dev/null || echo "skill-home: 已更新"
fi
echo ""
echo "备份保存在: $BACKUP_DIR"
