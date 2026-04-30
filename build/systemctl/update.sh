#!/bin/bash
set -e

# ------------------------------------------
# isrvd systemd 服务更新脚本
# ------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
SERVICE_NAME="isrvd"
BIN_FILE="/usr/local/bin/isrvd"

echo "=========================================="
echo "  isrvd Systemd Service Updater"
echo "=========================================="

# 检查是否以 root 运行
if [ "$EUID" -ne 0 ]; then
    echo "[error] Please run as root"
    exit 1
fi

# 查找二进制文件
BINARY_SRC=""
for f in "${PACKAGE_DIR}"/isrvd-*; do
    if [ -f "$f" ] && [ -x "$f" ]; then
        BINARY_SRC="$f"
        break
    fi
done

if [ -z "$BINARY_SRC" ]; then
    echo "[error] Binary not found in ${PACKAGE_DIR}"
    echo "[info] Please extract the package correctly"
    exit 1
fi

# 检查服务是否已安装
if [ ! -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
    echo "[error] Service not installed"
    echo "[info] Please run install.sh first"
    exit 1
fi

# 停止服务
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "[info] Stopping ${SERVICE_NAME} service..."
    systemctl stop "$SERVICE_NAME"
fi

# 更新二进制文件
echo "[info] Updating binary: ${BINARY_SRC} -> ${BIN_FILE}"
cp "$BINARY_SRC" "$BIN_FILE"
chmod +x "$BIN_FILE"

# 启动服务
echo "[info] Starting ${SERVICE_NAME} service..."
systemctl start "$SERVICE_NAME"

echo ""
echo "=========================================="
echo "  Update Complete!"
echo "=========================================="
echo ""
echo "  Binary: ${BIN_FILE}"
echo "  Config: /etc/isrvd/config.yml (preserved)"
echo ""
echo "=========================================="
