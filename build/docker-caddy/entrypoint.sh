#!/bin/bash
set -e

echo "=========================================="
echo "  caddy + isrvd  All-in-One               "
echo "=========================================="

# ------------------------------------------
# 配置文件自动填充
# ------------------------------------------

CONF_DIR="/data/conf"
DEFAULTS_DIR="/etc/defaults"

mkdir -p "$CONF_DIR"

# 遍历默认配置模板，不存在则自动拷贝
for file in "$DEFAULTS_DIR"/*; do
    filename=$(basename "$file")
    if [ ! -f "$CONF_DIR/$filename" ]; then
        echo "[init] Creating default config: $CONF_DIR/$filename"
        cp "$file" "$CONF_DIR/$filename"
    else
        echo "[init] Using existing config: $CONF_DIR/$filename"
    fi
done

# ------------------------------------------
# 确保关键配置文件确实存在
# ------------------------------------------

for required in isrvd.yml caddy.json; do
    if [ ! -f "$CONF_DIR/$required" ]; then
        echo "[error] Required config not found: $CONF_DIR/$required"
        exit 1
    fi
done

# ------------------------------------------
# 输出默认密码提示信息
# ------------------------------------------

if grep -q 'password: admin' "$CONF_DIR/isrvd.yml" 2>/dev/null; then
    echo "=========================================="
    echo "  首次启动 - 默认密码为 admin"
    echo "=========================================="
    echo "  isrvd 管理员用户: admin"
    echo "  isrvd 管理员密码: admin"
    echo ""
    echo "  请登录后立即修改密码"
    echo "=========================================="
fi

# ------------------------------------------
# 初始化 runit 服务
# ------------------------------------------

if [ ! -L /etc/service/caddy ]; then
    mkdir -p /etc/service
    find /etc/sv -type f \( -name run -o -name finish \) -exec chmod +x {} \;
    ln -s /etc/sv/caddy /etc/service/caddy
    ln -s /etc/sv/isrvd /etc/service/isrvd
fi

echo "[init] Starting all services via runit..."
exec /usr/sbin/runsvdir -P /etc/service
