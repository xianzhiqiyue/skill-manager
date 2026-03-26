#!/usr/bin/env bash

set -euo pipefail

VERSION="${1:-dev}"
DIST_DIR="${2:-dist}"
BINARY_NAME="skill-home"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

case "$DIST_DIR" in
  /*)
    DIST_ABS="$DIST_DIR"
    ;;
  *)
    DIST_ABS="${ROOT_DIR}/${DIST_DIR}"
    ;;
esac

COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
BUILD_DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
LDFLAGS="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

rm -rf "$DIST_ABS"
mkdir -p "$DIST_ABS"

package_unix() {
  local goos="$1"
  local goarch="$2"
  local asset_name="${BINARY_NAME}-${goos}-${goarch}.tar.gz"
  local work_dir="${TMP_DIR}/${goos}-${goarch}"

  mkdir -p "$work_dir"
  GOOS="$goos" GOARCH="$goarch" go build -ldflags "$LDFLAGS" -o "${work_dir}/${BINARY_NAME}" ./cmd/skill-home
  tar -C "$work_dir" -czf "${DIST_ABS}/${asset_name}" "$BINARY_NAME"
}

package_windows() {
  local goarch="$1"
  local asset_name="${BINARY_NAME}-windows-${goarch}.zip"
  local work_dir="${TMP_DIR}/windows-${goarch}"
  local asset_path="${DIST_ABS}/${asset_name}"

  mkdir -p "$work_dir"
  GOOS=windows GOARCH="$goarch" go build -ldflags "$LDFLAGS" -o "${work_dir}/${BINARY_NAME}.exe" ./cmd/skill-home
  if command -v zip >/dev/null 2>&1; then
    (
      cd "$work_dir"
      zip -q "$asset_path" "${BINARY_NAME}.exe"
    )
    return
  fi

  python3 - <<PY
import pathlib
import zipfile

work_dir = pathlib.Path(${work_dir@Q})
asset_path = pathlib.Path(${asset_path@Q})

with zipfile.ZipFile(asset_path, "w", compression=zipfile.ZIP_DEFLATED) as zf:
    zf.write(work_dir / "${BINARY_NAME}.exe", arcname="${BINARY_NAME}.exe")
PY
}

package_unix darwin amd64
package_unix darwin arm64
package_unix linux amd64
package_unix linux arm64
package_windows amd64

if command -v sha256sum >/dev/null 2>&1; then
  (
    cd "$DIST_ABS"
    sha256sum ./* > checksums.txt
  )
elif command -v shasum >/dev/null 2>&1; then
  (
    cd "$DIST_ABS"
    shasum -a 256 ./* > checksums.txt
  )
else
  echo "需要 sha256sum 或 shasum 来生成校验文件" >&2
  exit 1
fi

echo "Release assets created in ${DIST_ABS}/"
