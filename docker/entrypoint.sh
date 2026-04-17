#!/bin/bash
set -e

echo "=========================================="
echo "  Apisix + etcd + isrvd  All-in-One"
echo "=========================================="

# ------------------------------------------
# 配置文件自动填充
# 当用户挂载 /data 目录时，自动填充默认配置
# ------------------------------------------
CONF_DIR="/data/conf"
DEFAULTS_DIR="/etc/defaults"

mkdir -p "$CONF_DIR"
mkdir -p /var/log/supervisor
mkdir -p /data/conf /data/etcd /data/container /data/storage

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

# 生成随机 Apisix Admin Key（32 位十六进制）
if grep -rq '__Apisix_ADMIN_KEY__' "$CONF_DIR/" 2>/dev/null; then
    Apisix_ADMIN_KEY=$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')
    replace_placeholder '__Apisix_ADMIN_KEY__' "$Apisix_ADMIN_KEY"
    echo "[init] Generated random Apisix admin key"
fi

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
# 将 Apisix 配置链接到标准位置
# ------------------------------------------
ln -sf "$CONF_DIR/apisix.yaml" /usr/local/apisix/conf/config.yaml

# 确保关键配置文件确实存在
for required in apisix.yaml isrvd.yml; do
    if [ ! -f "$CONF_DIR/$required" ]; then
        echo "[error] Required config not found: $CONF_DIR/$required"
        exit 1
    fi
done

# ------------------------------------------
# 初始化 Apisix（需要 etcd 就绪）
# ------------------------------------------
cd /usr/local/apisix
/usr/local/openresty/luajit/bin/luajit apisix/cli/apisix.lua init

# 检测 etcd 数据目录判断集群状态
if [ -d "/data/etcd/member" ]; then
    ETCD_CLUSTER_STATE="existing"
else
    ETCD_CLUSTER_STATE="new"
fi
echo "[init] etcd cluster state: $ETCD_CLUSTER_STATE"

# 临时启动 etcd，等待就绪后执行 init_etcd
echo "[init] Starting temporary etcd for Apisix init..."
/usr/local/bin/etcd \
    --name=apisix-etcd \
    --data-dir=/data/etcd \
    --listen-client-urls=http://0.0.0.0:2379 \
    --advertise-client-urls=http://0.0.0.0:2379 \
    --listen-peer-urls=http://0.0.0.0:2380 \
    --initial-advertise-peer-urls=http://0.0.0.0:2380 \
    --initial-cluster=apisix-etcd=http://0.0.0.0:2380 \
    --initial-cluster-token=apisix-etcd-cluster \
    --initial-cluster-state="$ETCD_CLUSTER_STATE" \
    &>/dev/null &
ETCD_PID=$!

# 等待 etcd 就绪（最多 30 秒）
for i in $(seq 1 30); do
    if etcdctl endpoint health --endpoints=http://127.0.0.1:2379 &>/dev/null; then
        echo "[init] etcd is ready"
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "[error] etcd failed to start within 30s"
        kill $ETCD_PID 2>/dev/null
        exit 1
    fi
    sleep 1
done

/usr/local/openresty/luajit/bin/luajit apisix/cli/apisix.lua init_etcd

# 停掉临时 etcd，后续由 supervisord 管理
echo "[init] Stopping temporary etcd..."
kill $ETCD_PID 2>/dev/null
wait $ETCD_PID 2>/dev/null || true

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

echo "[init] Starting all services via supervisord..."
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
