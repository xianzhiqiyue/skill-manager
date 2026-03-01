#!/bin/bash

# skill-home CLI 安装脚本

set -e

REPO="skill-home/cli"
BINARY_NAME="skill-home"
INSTALL_DIR="/usr/local/bin"

# 检测操作系统和架构
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            echo "不支持的架构: $ARCH"
            exit 1
            ;;
    esac

    case $OS in
        linux|darwin)
            PLATFORM="${OS}-${ARCH}"
            ;;
        mingw*|msys*|cygwin*)
            PLATFORM="windows-${ARCH}.exe"
            ;;
        *)
            echo "不支持的操作系统: $OS"
            exit 1
            ;;
    esac

    echo "$PLATFORM"
}

# 获取最新版本
get_latest_version() {
    curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# 下载并安装
download_and_install() {
    PLATFORM=$1
    VERSION=$2

    if [ -z "$VERSION" ]; then
        echo "正在获取最新版本..."
        VERSION=$(get_latest_version)
        if [ -z "$VERSION" ]; then
            echo "无法获取最新版本，请检查网络连接"
            exit 1
        fi
    fi

    echo "安装版本: $VERSION"
    echo "平台: $PLATFORM"

    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}-${PLATFORM}"

    # 创建临时目录
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    echo "正在下载..."
    if command -v curl &> /dev/null; then
        curl -L -o "${TMP_DIR}/${BINARY_NAME}" "$DOWNLOAD_URL"
    elif command -v wget &> /dev/null; then
        wget -O "${TMP_DIR}/${BINARY_NAME}" "$DOWNLOAD_URL"
    else
        echo "需要 curl 或 wget 来下载"
        exit 1
    fi

    # 检查下载是否成功
    if [ ! -f "${TMP_DIR}/${BINARY_NAME}" ]; then
        echo "下载失败"
        exit 1
    fi

    # 添加执行权限
    chmod +x "${TMP_DIR}/${BINARY_NAME}"

    # 安装到指定目录
    echo "安装到 ${INSTALL_DIR}..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        echo "需要管理员权限来安装到 ${INSTALL_DIR}"
        sudo mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    echo "安装完成!"
    echo ""
    echo "运行 'skill-home --help' 开始使用"
}

# 主函数
main() {
    echo "================================"
    echo "  skill-home CLI 安装脚本"
    echo "================================"
    echo ""

    PLATFORM=$(detect_platform)
    VERSION="${1:-}"

    download_and_install "$PLATFORM" "$VERSION"
}

main "$@"
