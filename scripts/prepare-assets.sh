#!/usr/bin/env bash
# prepare-assets.sh 为 GoReleaser 准备发布资源。
#
# 1. clone / 更新 go-ocr-model 仓库（包含 ddddocr 模型与字典）。
# 2. 按平台下载 ONNX Runtime 动态库，并整理到 go-ocr-model/lib/<os>/<arch>/。
#
# 环境变量：
#   ONNXRUNTIME_VERSION - ONNX Runtime 版本，默认 1.19.2
#   GO_OCR_MODEL_REPO   - 模型仓库地址，默认 https://github.com/muzimu/go-ocr-model.git

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

ONNX_VERSION="${ONNXRUNTIME_VERSION:-1.19.2}"
MODEL_REPO="${GO_OCR_MODEL_REPO:-https://github.com/muzimu/go-ocr-model.git}"
MODEL_DIR="go-ocr-model"
LIB_DIR="$MODEL_DIR/lib"

echo "== Preparing release assets =="

# 1. 准备模型文件
if [ ! -d "$MODEL_DIR/ddddocr" ]; then
  echo "Cloning $MODEL_REPO ..."
  rm -rf "$MODEL_DIR"
  git clone --depth 1 "$MODEL_REPO" "$MODEL_DIR"
else
  echo "Model directory already exists: $MODEL_DIR/ddddocr"
fi

# 2. 准备平台相关的 ONNX Runtime 库
mkdir -p "$LIB_DIR/darwin/amd64"
mkdir -p "$LIB_DIR/darwin/arm64"
mkdir -p "$LIB_DIR/linux/amd64"
mkdir -p "$LIB_DIR/linux/arm64"
mkdir -p "$LIB_DIR/windows/amd64"

# 下载并解压到临时目录，然后复制指定文件到目标路径。
# 参数：url dest_file find_name
download_and_copy() {
  local url="$1"
  local dest_file="$2"
  local find_name="$3"

  if [ -f "$dest_file" ]; then
    echo "  exists: $dest_file"
    return 0
  fi

  local tmpdir
  tmpdir=$(mktemp -d)
  local archive="$tmpdir/archive"

  echo "  downloading: $url"
  curl -fsSL --retry 3 --retry-delay 2 -o "$archive" "$url"

  case "$url" in
    *.zip)
      unzip -q "$archive" -d "$tmpdir/extract"
      ;;
    *.tgz|*.tar.gz)
      mkdir -p "$tmpdir/extract"
      tar -xzf "$archive" -C "$tmpdir/extract"
      ;;
    *)
      echo "Unsupported archive format: $url" >&2
      exit 1
      ;;
  esac

  local src_file
  src_file=$(find "$tmpdir/extract" -name "$find_name" -type f | head -n 1)
  if [ -z "$src_file" ]; then
    echo "ERROR: could not find '$find_name' in $url" >&2
    find "$tmpdir/extract" -type f >&2
    exit 1
  fi

  cp "$src_file" "$dest_file"
  echo "  copied: $dest_file"

  rm -rf "$tmpdir"
}

# macOS Apple Silicon
download_and_copy \
  "https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/onnxruntime-osx-arm64-${ONNX_VERSION}.tgz" \
  "$LIB_DIR/darwin/arm64/onnxruntime_arm64.dylib" \
  "libonnxruntime.*.dylib"

# macOS Intel
download_and_copy \
  "https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/onnxruntime-osx-x86_64-${ONNX_VERSION}.tgz" \
  "$LIB_DIR/darwin/amd64/onnxruntime_amd64.dylib" \
  "libonnxruntime.*.dylib"

# Linux AMD64
download_and_copy \
  "https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/onnxruntime-linux-x64-${ONNX_VERSION}.tgz" \
  "$LIB_DIR/linux/amd64/onnxruntime_amd64.so" \
  "libonnxruntime.so.*"

# Linux ARM64
download_and_copy \
  "https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/onnxruntime-linux-aarch64-${ONNX_VERSION}.tgz" \
  "$LIB_DIR/linux/arm64/onnxruntime_arm64.so" \
  "libonnxruntime.so.*"

# Windows AMD64
download_and_copy \
  "https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/onnxruntime-win-x64-${ONNX_VERSION}.zip" \
  "$LIB_DIR/windows/amd64/onnxruntime.dll" \
  "onnxruntime.dll"

echo "== Assets ready =="
