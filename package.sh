#!/bin/bash

set -e

PROJECT_NAME="k8sloadsecure"
VERSION="1.0.0"
BUILD_DATE=$(date +%Y%m%d_%H%M%S)
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_DIRTY=$(git diff --quiet 2>/dev/null && echo "" || echo "-dirty")
OUTPUT_DIR="$(cd "$(dirname "$0")" && pwd)/release"
PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"

mkdir -p "$OUTPUT_DIR"

echo "======================================"
echo "  K8sGate 构建打包脚本"
echo "======================================"
echo ""
echo "  版本:  ${VERSION}"
echo "  提交:  ${GIT_COMMIT}${GIT_DIRTY}"
echo "  时间:  ${BUILD_DATE}"
echo ""

# ----------------------------------------
# 1. 编译后端 (交叉编译 linux/amd64)
# ----------------------------------------
echo "[1/4] 编译后端 (linux/amd64)..."
cd "$PROJECT_DIR/backend"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT}${GIT_DIRTY} -X main.BuildDate=${BUILD_DATE}" \
    -o k8sgate-backend .

echo "  二进制: $(du -sh k8sgate-backend | awk '{print $1}')"
file k8sgate-backend | grep -q "ELF 64-bit" || { echo "  [ERROR] 不是 linux/amd64 二进制!"; exit 1; }
echo "  架构验证: OK (ELF 64-bit x86-64)"

# ----------------------------------------
# 2. 编译前端
# ----------------------------------------
echo ""
echo "[2/4] 编译前端..."
cd "$PROJECT_DIR/frontend"
npm run build --silent 2>&1 | tail -3
echo "  dist: $(du -sh dist | awk '{print $1}')"

# ----------------------------------------
# 3. 打包（保持目录结构）
# ----------------------------------------
echo ""
echo "[3/4] 打包..."

PACKAGE_NAME="${PROJECT_NAME}-${VERSION}-${GIT_COMMIT}${GIT_DIRTY}-${BUILD_DATE}.tar.gz"
PACKAGE_PATH="${OUTPUT_DIR}/${PACKAGE_NAME}"

TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# 创建与 deploy-test.sh 期望一致的目录结构
STAGE="${TMP_DIR}/${PROJECT_NAME}"
mkdir -p "${STAGE}/backend"
mkdir -p "${STAGE}/frontend"
mkdir -p "${STAGE}/deploy"

# 后端：只放编译产物
cp "$PROJECT_DIR/backend/k8sgate-backend" "${STAGE}/backend/"

# 前端：只放构建产物
cp -r "$PROJECT_DIR/frontend/dist" "${STAGE}/frontend/dist"

# 部署文件
cp "$PROJECT_DIR/deploy/deploy-test.sh" "${STAGE}/deploy/"
cp "$PROJECT_DIR/deploy/init.sql" "${STAGE}/deploy/"
cp "$PROJECT_DIR/deploy/nginx.conf" "${STAGE}/deploy/" 2>/dev/null || true

# 写入版本信息
cat > "${STAGE}/VERSION" <<EOF
version: ${VERSION}
git_commit: ${GIT_COMMIT}${GIT_DIRTY}
build_date: ${BUILD_DATE}
build_host: $(hostname)
EOF

cd "$TMP_DIR"
tar -czf "$PACKAGE_PATH" "${PROJECT_NAME}/"

SIZE=$(du -sh "$PACKAGE_PATH" | awk '{print $1}')

# ----------------------------------------
# 4. 验证包内容
# ----------------------------------------
echo ""
echo "[4/4] 验证包内容..."
echo "  包含文件:"
tar -tzf "$PACKAGE_PATH" | head -15
echo "  ..."

# 更新 latest 链接
ln -sf "$PACKAGE_NAME" "${OUTPUT_DIR}/${PROJECT_NAME}-latest.tar.gz"

echo ""
echo "======================================"
echo "  打包完成"
echo "======================================"
echo ""
echo "  文件: ${PACKAGE_PATH}"
echo "  大小: ${SIZE}"
echo "  提交: ${GIT_COMMIT}${GIT_DIRTY}"
echo ""
echo "  部署命令:"
echo "    make deploy-test"
echo ""
