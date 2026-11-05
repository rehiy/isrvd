#!/bin/bash
set -e

echo "=========================================="
echo "  caddy + isrvd  All-in-One               "
echo "=========================================="

# ------------------------------------------
# 配置文件自动填充
# ------------------------------------------

CONF_DIR="/data/conf"
CADDY_DIR="/data/caddy"
CADDY_CONF="$CADDY_DIR/caddy.json"
DEFAULTS_DIR="/etc/defaults"

mkdir -p "$CONF_DIR" "$CADDY_DIR"

# 旧版布局 conf/caddy.json -> caddy/caddy.json（只复制，保留旧文件以便回滚）
if [ -f "$CONF_DIR/caddy.json" ] && [ ! -e "$CADDY_CONF" ]; then
    echo "[init] Migrating legacy config: $CONF_DIR/caddy.json -> $CADDY_CONF"
    cp -p "$CONF_DIR/caddy.json" "$CADDY_CONF"
fi

# 默认配置模板不存在则自动拷贝
install_default() {
    local src="$1" dest="$2"
    if [ ! -f "$dest" ]; then
        echo "[init] Creating default config: $dest"
        cp "$src" "$dest"
    else
        echo "[init] Using existing config: $dest"
    fi
}
install_default "$DEFAULTS_DIR/isrvd.yml" "$CONF_DIR/isrvd.yml"
install_default "$DEFAULTS_DIR/caddy.json" "$CADDY_CONF"

# ------------------------------------------
# 确保关键配置文件确实存在
# ------------------------------------------

for required in "$CONF_DIR/isrvd.yml" "$CADDY_CONF"; do
    if [ ! -f "$required" ]; then
        echo "[error] Required config not found: $required"
        exit 1
    fi
done
chmod 600 "$CONF_DIR/isrvd.yml" "$CADDY_CONF"

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
