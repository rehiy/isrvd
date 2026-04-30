#!/bin/bash
set -e

# ------------------------------------------
# isrvd systemd 服务卸载脚本
# ------------------------------------------

SERVICE_NAME="isrvd"

echo "=========================================="
echo "  isrvd Systemd Service Uninstaller"
echo "=========================================="

# 检查是否以 root 运行
if [ "$EUID" -ne 0 ]; then
    echo "[error] Please run as root"
    exit 1
fi

# 停止服务
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "[info] Stopping ${SERVICE_NAME} service..."
    systemctl stop "$SERVICE_NAME"
fi

# 禁用服务
if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
    echo "[info] Disabling ${SERVICE_NAME} service..."
    systemctl disable "$SERVICE_NAME"
fi

# 删除服务单元文件
if [ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
    echo "[info] Removing service file..."
    rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
fi

# 删除二进制文件
BIN_FILE="/usr/local/bin/isrvd"
if [ -f "$BIN_FILE" ]; then
    echo "[info] Removing binary: $BIN_FILE"
    rm -f "$BIN_FILE"
fi

# 重新加载 systemd
echo "[info] Reloading systemd daemon..."
systemctl daemon-reload

echo ""
echo "=========================================="
echo "  Uninstallation Complete!"
echo "=========================================="
echo ""
echo "  Note: Config files in /etc/isrvd are preserved"
echo ""