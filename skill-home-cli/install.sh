#!/usr/bin/env bash

set -euo pipefail

BINARY_NAME="skill-home"
INSTALL_DIR="${SKILL_HOME_INSTALL_DIR:-${HOME}/.local/bin}"
DEFAULT_RELEASES_BASE_URL="__SKILL_HOME_RELEASES_BASE_URL__"
if [[ "$DEFAULT_RELEASES_BASE_URL" == "__SKILL_HOME_RELEASES_BASE_URL__" ]]; then
  DEFAULT_RELEASES_BASE_URL="http://47.122.112.210:8080/releases"
fi
HOSTED_RELEASES_BASE_URL="${SKILL_HOME_RELEASES_BASE_URL:-$DEFAULT_RELEASES_BASE_URL}"

detect_platform() {
  local os arch

  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"

  case "$arch" in
    x86_64)
      arch="amd64"
      ;;
    arm64|aarch64)
      arch="arm64"
      ;;
    *)
      echo "不支持的架构: ${arch}" >&2
      exit 1
      ;;
  esac

  case "$os" in
    linux|darwin)
      PLATFORM="${os}-${arch}"
      ARCHIVE_EXT="tar.gz"
      BINARY_FILENAME="${BINARY_NAME}"
      ;;
    mingw*|msys*|cygwin*)
      PLATFORM="windows-${arch}"
      ARCHIVE_EXT="zip"
      BINARY_FILENAME="${BINARY_NAME}.exe"
      ;;
    *)
      echo "不支持的操作系统: ${os}" >&2
      exit 1
      ;;
  esac
}

try_download_file() {
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

  echo "需要 curl 或 wget 来下载安装包" >&2
  exit 1
}

get_latest_version_from_hosted() {
  local tmp_file version

  tmp_file="$(mktemp)"

  if ! try_download_file "${HOSTED_RELEASES_BASE_URL%/}/latest.json" "$tmp_file"; then
    rm -f "$tmp_file"
    return 1
  fi

  version="$(grep '"tag_name":' "$tmp_file" | head -n1 | sed -E 's/.*"([^"]+)".*/\1/')"
  rm -f "$tmp_file"
  [ -n "$version" ] || return 1

  echo "$version"
}

get_latest_version() {
  get_latest_version_from_hosted
}

normalize_version() {
  local version="${1:-}"

  if [ -z "$version" ]; then
    echo "正在获取最新版本..." >&2
    version="$(get_latest_version)"
    if [ -z "$version" ]; then
      echo "无法获取最新版本，请确认 Skill Home 服务已同步该 CLI 发布" >&2
      exit 1
    fi
  fi

  if [[ "$version" != v* ]]; then
    version="v${version}"
  fi

  echo "$version"
}

verify_checksum() {
  local archive_path="$1"
  local checksums_path="$2"
  local asset_name="$3"

  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$(dirname "$checksums_path")" && sha256sum -c --ignore-missing "$(basename "$checksums_path")" 2>/dev/null | grep -F "$(basename "$archive_path"): OK" >/dev/null) && return
  elif command -v shasum >/dev/null 2>&1; then
    local expected actual
    expected="$(grep " ${asset_name}\$" "$checksums_path" | awk '{print $1}')"
    actual="$(shasum -a 256 "$archive_path" | awk '{print $1}')"
    [ -n "$expected" ] && [ "$expected" = "$actual" ] && return
  else
    echo "未找到 sha256sum/shasum，跳过 checksum 校验" >&2
    return
  fi

  echo "checksum 校验失败: ${asset_name}" >&2
  return 1
}

extract_archive() {
  local archive_path="$1"
  local archive_ext="$2"
  local target_dir="$3"

  case "$archive_ext" in
    tar.gz)
      tar -xzf "$archive_path" -C "$target_dir"
      ;;
    zip)
      unzip -q "$archive_path" -d "$target_dir"
      ;;
    *)
      echo "不支持的归档格式: ${archive_ext}" >&2
      exit 1
      ;;
  esac
}

install_binary() {
  local source_path="$1"

  mkdir -p "$INSTALL_DIR"
  install -m 0755 "$source_path" "${INSTALL_DIR}/${BINARY_FILENAME}"
}

show_path_hint() {
  case ":${PATH}:" in
    *":${INSTALL_DIR}:"*)
      ;;
    *)
      echo ""
      echo "请将 ${INSTALL_DIR} 加入 PATH，例如："
      echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
      ;;
  esac
}

build_hosted_release_url() {
  local version="$1"
  local asset_name="$2"

  echo "${HOSTED_RELEASES_BASE_URL%/}/${version}/${asset_name}"
}

download_release_bundle() {
  local version="$1"
  local asset_name="$2"
  local archive_path="$3"
  local checksums_path="$4"
  local release_url checksum_url

  release_url="$(build_hosted_release_url "$version" "$asset_name")"
  checksum_url="$(build_hosted_release_url "$version" "checksums.txt")"

  try_download_file "$release_url" "$archive_path" || {
    echo "Skill Home 发布包下载失败" >&2
    return 1
  }

  try_download_file "$checksum_url" "$checksums_path" || {
    echo "Skill Home 校验文件下载失败" >&2
    return 1
  }

  verify_checksum "$archive_path" "$checksums_path" "$asset_name" || {
    echo "Skill Home checksum 校验失败" >&2
    return 1
  }
}

main() {
  echo "================================"
  echo "  skill-home CLI 安装脚本"
  echo "================================"
  echo ""

  detect_platform
  VERSION="$(normalize_version "${1:-}")"
  ASSET_NAME="${BINARY_NAME}-${PLATFORM}.${ARCHIVE_EXT}"

  echo "版本: ${VERSION}"
  echo "平台: ${PLATFORM}"
  echo "安装目录: ${INSTALL_DIR}"
  echo "Skill Home 发布源: ${HOSTED_RELEASES_BASE_URL%/}"
  echo ""

  TMP_DIR="$(mktemp -d)"
  trap 'rm -rf "${TMP_DIR}"' EXIT

  ARCHIVE_PATH="${TMP_DIR}/${ASSET_NAME}"
  CHECKSUMS_PATH="${TMP_DIR}/checksums.txt"
  EXTRACT_DIR="${TMP_DIR}/extract"
  mkdir -p "$EXTRACT_DIR"

  echo "正在下载发布包..."
  if ! download_release_bundle "$VERSION" "$ASSET_NAME" "$ARCHIVE_PATH" "$CHECKSUMS_PATH"; then
    exit 1
  fi

  echo "正在解压..."
  extract_archive "$ARCHIVE_PATH" "$ARCHIVE_EXT" "$EXTRACT_DIR"

  BINARY_PATH="${EXTRACT_DIR}/${BINARY_FILENAME}"
  if [ ! -f "$BINARY_PATH" ]; then
    echo "安装包内未找到 ${BINARY_FILENAME}" >&2
    exit 1
  fi

  echo "正在安装..."
  install_binary "$BINARY_PATH"

  echo ""
  echo "安装完成: ${INSTALL_DIR}/${BINARY_FILENAME}"
  echo "运行 '${BINARY_NAME} --help' 开始使用"
  show_path_hint
}

main "$@"
