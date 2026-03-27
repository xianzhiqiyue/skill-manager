#!/bin/bash
# Skill-Home 更新部署脚本
# 在阿里云服务器上执行

set -e

echo "=== Skill-Home 更新部署 ==="

DEPLOY_DIR="/opt/skill-home"
BACKUP_DIR="/opt/skill-home/backup-$(date +%Y%m%d-%H%M%S)"

# 创建备份目录
mkdir -p "$BACKUP_DIR"

# 备份现有版本
echo "[1/5] 备份现有版本..."
if [ -f "$DEPLOY_DIR/server" ]; then
    cp "$DEPLOY_DIR/server" "$BACKUP_DIR/server"
fi
if [ -f "$DEPLOY_DIR/skill-home" ]; then
    cp "$DEPLOY_DIR/skill-home" "$BACKUP_DIR/skill-home"
fi
if [ -f "$DEPLOY_DIR/install.sh" ]; then
    cp "$DEPLOY_DIR/install.sh" "$BACKUP_DIR/install.sh"
fi
if [ -d "$DEPLOY_DIR/releases" ]; then
    cp -a "$DEPLOY_DIR/releases" "$BACKUP_DIR/releases"
fi

# 停止服务
echo "[2/5] 停止 API 服务..."
systemctl stop skill-home || true

# 更新文件
echo "[3/5] 更新二进制文件..."
if [ -f "$DEPLOY_DIR/server.new" ]; then
    mv "$DEPLOY_DIR/server" "$DEPLOY_DIR/server.old" 2>/dev/null || true
    mv "$DEPLOY_DIR/server.new" "$DEPLOY_DIR/server"
    chmod +x "$DEPLOY_DIR/server"
fi

if [ -f "$DEPLOY_DIR/skill-home.new" ]; then
    mv "$DEPLOY_DIR/skill-home" "$DEPLOY_DIR/skill-home.old" 2>/dev/null || true
    mv "$DEPLOY_DIR/skill-home.new" "$DEPLOY_DIR/skill-home"
    chmod +x "$DEPLOY_DIR/skill-home"
fi

if [ -f "$DEPLOY_DIR/install.sh.new" ]; then
    mv "$DEPLOY_DIR/install.sh" "$DEPLOY_DIR/install.sh.old" 2>/dev/null || true
    mv "$DEPLOY_DIR/install.sh.new" "$DEPLOY_DIR/install.sh"
    chmod +x "$DEPLOY_DIR/install.sh"
fi

if [ -d "$DEPLOY_DIR/releases.new" ]; then
    rm -rf "$DEPLOY_DIR/releases.old"
    mv "$DEPLOY_DIR/releases" "$DEPLOY_DIR/releases.old" 2>/dev/null || true
    mv "$DEPLOY_DIR/releases.new" "$DEPLOY_DIR/releases"
fi

# 启动服务
echo "[4/5] 启动 API 服务..."
systemctl start skill-home

# 健康检查
echo "[5/5] 健康检查..."
sleep 2
if curl -s http://localhost:8080/health > /dev/null \
    && curl -fsSL http://localhost:8080/install.sh | grep -q "skill-home CLI 安装脚本" \
    && { [ ! -d "$DEPLOY_DIR/releases" ] || curl -fsSL http://localhost:8080/releases/latest.json | grep -q '"tag_name"'; }; then
    echo "✅ 部署成功！服务运行正常"
    echo ""
    echo "版本信息:"
    "$DEPLOY_DIR/server" --version 2>/dev/null || echo "server: 已更新"
    "$DEPLOY_DIR/skill-home" --version 2>/dev/null || echo "skill-home: 已更新"
else
    echo "❌ 健康检查失败，正在回滚..."
    systemctl stop skill-home
    mv "$DEPLOY_DIR/server.old" "$DEPLOY_DIR/server"
    mv "$DEPLOY_DIR/skill-home.old" "$DEPLOY_DIR/skill-home"
    if [ -f "$DEPLOY_DIR/install.sh.old" ]; then
        mv "$DEPLOY_DIR/install.sh.old" "$DEPLOY_DIR/install.sh"
    fi
    if [ -d "$DEPLOY_DIR/releases.old" ]; then
        rm -rf "$DEPLOY_DIR/releases"
        mv "$DEPLOY_DIR/releases.old" "$DEPLOY_DIR/releases"
    fi
    systemctl start skill-home
    echo "✅ 已回滚到之前的版本"
    exit 1
fi

echo ""
echo "备份保存在: $BACKUP_DIR"
