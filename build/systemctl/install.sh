#!/bin/bash
set -e

# ------------------------------------------
# isrvd systemd 服务安装脚本
# ------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
SERVICE_NAME="isrvd"
SERVICE_FILE="${SCRIPT_DIR}/${SERVICE_NAME}.service"
CONFIG_DIR="/etc/isrvd"
BIN_FILE="/usr/local/bin/isrvd"

echo "=========================================="
echo "  isrvd Systemd Service Installer"
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

# 安装二进制文件
echo "[info] Installing binary: ${BINARY_SRC} -> ${BIN_FILE}"
cp "$BINARY_SRC" "$BIN_FILE"
chmod +x "$BIN_FILE"

# 创建配置目录
if [ ! -d "$CONFIG_DIR" ]; then
    echo "[info] Creating directory: $CONFIG_DIR"
    mkdir -p "$CONFIG_DIR"
fi

# 创建默认配置文件（如果不存在）
if [ ! -f "${CONFIG_DIR}/config.yml" ]; then
    if [ -f "${PACKAGE_DIR}/config.yml" ]; then
        echo "[info] Copying default config to ${CONFIG_DIR}/config.yml"
        cp "${PACKAGE_DIR}/config.yml" "${CONFIG_DIR}/config.yml"
    else
        echo "[warn] Default config not found, please create ${CONFIG_DIR}/config.yml manually"
    fi
fi

# 安装服务单元文件
echo "[info] Installing systemd service..."
cp "$SERVICE_FILE" /etc/systemd/system/

# 重新加载 systemd
echo "[info] Reloading systemd daemon..."
systemctl daemon-reload

# 启用服务
echo "[info] Enabling ${SERVICE_NAME} service..."
systemctl enable "$SERVICE_NAME"

echo ""
echo "=========================================="
echo "  Installation Complete!"
echo "=========================================="
echo ""
echo "  Config file: ${CONFIG_DIR}/config.yml"
echo ""
echo "  Commands:"
echo "    Start:   systemctl start ${SERVICE_NAME}"
echo "    Stop:    systemctl stop ${SERVICE_NAME}"
echo "    Restart: systemctl restart ${SERVICE_NAME}"
echo "    Status:  systemctl status ${SERVICE_NAME}"
echo "    Logs:    journalctl -u ${SERVICE_NAME} -f"
echo ""
echo "=========================================="