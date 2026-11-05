#!/bin/bash
set -e

echo "=========================================="
echo "  apisix + etcd + isrvd  All-in-One       "
echo "=========================================="

# ------------------------------------------
# 配置文件自动填充
# ------------------------------------------

CONF_DIR="/data/conf"
APISIX_DIR="/data/apisix"
APISIX_CONF="$APISIX_DIR/apisix.yaml"
DEFAULTS_DIR="/etc/defaults"

mkdir -p "$CONF_DIR" "$APISIX_DIR" /data/etcd

# 旧版布局 conf/apisix.yaml -> apisix/apisix.yaml（只复制，保留旧文件以便回滚）
if [ -f "$CONF_DIR/apisix.yaml" ] && [ ! -e "$APISIX_CONF" ]; then
    echo "[init] Migrating legacy config: $CONF_DIR/apisix.yaml -> $APISIX_CONF"
    cp -p "$CONF_DIR/apisix.yaml" "$APISIX_CONF"
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
install_default "$DEFAULTS_DIR/apisix.yaml" "$APISIX_CONF"

# ------------------------------------------
# 随机生成密钥（仅首次生成配置时替换占位符）
# ------------------------------------------

replace_placeholder() {
    local placeholder="$1"
    local value="$2"
    for file in "$CONF_DIR/isrvd.yml" "$APISIX_CONF"; do
        if grep -q "$placeholder" "$file" 2>/dev/null; then
            sed -i "s/$placeholder/$value/g" "$file"
            echo "[init] Replaced $placeholder in $(basename "$file")"
        fi
    done
}

# 生成随机 Apisix Admin Key（32 位十六进制）
if grep -q '__APISIX_ADMIN_KEY__' "$CONF_DIR/isrvd.yml" "$APISIX_CONF" 2>/dev/null; then
    APISIX_ADMIN_KEY=$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')
    replace_placeholder '__APISIX_ADMIN_KEY__' "$APISIX_ADMIN_KEY"
    echo "[init] Generated random Apisix admin key"
fi


# ------------------------------------------
# 确保关键配置文件确实存在
# ------------------------------------------

for required in "$APISIX_CONF" "$CONF_DIR/isrvd.yml"; do
    if [ ! -f "$required" ]; then
        echo "[error] Required config not found: $required"
        exit 1
    fi
done

for dir in conf cert logs; do
    target="/data/apisix/$dir"
    mkdir -p "$target"
    if [ ! -L "/usr/local/apisix/$dir" ]; then
        if [ -d "/usr/local/apisix/$dir" ]; then
            cp -an "/usr/local/apisix/$dir/." "$target/"
            rm -rf "/usr/local/apisix/$dir"
        fi
        ln -s "$target" "/usr/local/apisix/$dir"
    fi
done
ln -sf "$APISIX_CONF" /data/apisix/conf/config.yaml
chmod 600 "$CONF_DIR/isrvd.yml" "$APISIX_CONF"

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

if [ ! -L /etc/service/apisix ]; then
    find /etc/sv -type f \( -name run -o -name finish \) -exec chmod +x {} \;
    ln -s /etc/sv/apisix /etc/service/apisix
    ln -s /etc/sv/etcd /etc/service/etcd
    ln -s /etc/sv/isrvd /etc/service/isrvd
fi

echo "[init] Starting all services via runit..."
exec /usr/bin/runsvdir -P /etc/service
