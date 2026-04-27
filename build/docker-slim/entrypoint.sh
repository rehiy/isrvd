#!/bin/sh
set -e

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
mkdir -p /data/conf

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
# 随机生成密钥（仅首次生成配置时替换占位符）
# ------------------------------------------
replace_placeholder() {
    local placeholder="$1"
    local value="$2"
    for file in "$CONF_DIR"/*; do
        if grep -q "$placeholder" "$file" 2>/dev/null; then
            sed -i "s/$placeholder/$value/g" "$file"
            echo "[init] Replaced $placeholder in $(basename "$file")"
        fi
    done
}

# 生成随机 JWT Secret（32 位十六进制）
if grep -rq '__JWT_SECRET__' "$CONF_DIR/" 2>/dev/null; then
    JWT_SECRET=$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')
    replace_placeholder '__JWT_SECRET__' "$JWT_SECRET"
    echo "[init] Generated random JWT secret"
fi

# 生成随机 isrvd 管理员密码（12 位字母数字）
if grep -rq '__ISRVD_PASSWORD__' "$CONF_DIR/" 2>/dev/null; then
    ISRVD_PASSWORD=$(head -c 12 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 12)
    replace_placeholder '__ISRVD_PASSWORD__' "$ISRVD_PASSWORD"
    echo "[init] Generated random isrvd admin password"
fi

# ------------------------------------------
# 确保关键配置文件确实存在
# ------------------------------------------
for required in isrvd.yml; do
    if [ ! -f "$CONF_DIR/$required" ]; then
        echo "[error] Required config not found: $CONF_DIR/$required"
        exit 1
    fi
done

# ------------------------------------------
# 输出生成的密码信息（仅首次生成时显示）
# ------------------------------------------
if [ -n "$ISRVD_PASSWORD" ]; then
    echo "=========================================="
    echo "  首次启动 - 已生成随机密码"
    echo "=========================================="
    echo "  isrvd 管理员用户: admin"
    echo "  isrvd 管理员密码: $ISRVD_PASSWORD"
    echo ""
    echo "  密码已写入: $CONF_DIR/isrvd.yml"
    echo "  如需修改请编辑该文件后重启容器"
    echo "=========================================="
fi

echo "[init] Starting isrvd..."
exec /usr/local/bin/isrvd
