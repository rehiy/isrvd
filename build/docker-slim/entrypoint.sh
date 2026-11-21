#!/bin/sh
set -e

. /usr/local/lib/isrvd/entrypoint-functions.sh

echo "=========================================="
echo "  isrvd  Slim                             "
echo "=========================================="

# ------------------------------------------
# 配置文件自动填充
# 当用户挂载 /data 目录时，自动填充默认配置
# ------------------------------------------
CONF_DIR="/data/conf"
DEFAULTS_DIR="/etc/defaults"

mkdir -p "$CONF_DIR"
# 遍历默认配置模板，不存在则自动拷贝
for file in "$DEFAULTS_DIR"/*; do
    install_default "$file" "$CONF_DIR/${file##*/}"
done

# ------------------------------------------
# 确保关键配置文件确实存在
# ------------------------------------------
for required in isrvd.yml; do
    if [ ! -f "$CONF_DIR/$required" ]; then
        echo "[error] Required config not found: $CONF_DIR/$required"
        exit 1
    fi
done
chmod 600 "$CONF_DIR/isrvd.yml"

# ------------------------------------------
# 输出默认密码提示信息
# ------------------------------------------
print_default_password_notice "$CONF_DIR/isrvd.yml"

echo "[init] Using ISRVD_CONFIG: $ISRVD_CONFIG"
echo "[init] Starting isrvd..."
exec /usr/local/bin/isrvd
