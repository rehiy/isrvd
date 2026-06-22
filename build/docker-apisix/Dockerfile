###############################################
# 阶段 1: 下载 isrvd、etcd 和 lua-resty-openssl
###############################################
FROM alpine:3 AS builder

ARG TARGETARCH
ARG ISRVD_VERSION
ARG ISRVD_DOWNLOAD_BASE=https://github.com/rehiy/isrvd/releases/download
ARG ETCD_VERSION=v3.5.29
ARG LUA_RESTY_OPENSSL_VERSION=master

RUN apk add --no-cache curl tar \
    && isrvd_download_url="${ISRVD_DOWNLOAD_BASE}/${ISRVD_VERSION}/isrvd-linux-${TARGETARCH}.tar.gz" \
    && curl -fsSL "${isrvd_download_url}" \
       | tar xz -C /usr/local/bin/ isrvd-linux-${TARGETARCH} \
    && mv /usr/local/bin/isrvd-linux-${TARGETARCH} /usr/local/bin/isrvd \
    && curl -fsSL "https://github.com/etcd-io/etcd/releases/download/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-${TARGETARCH}.tar.gz" \
       | tar xz --strip-components=1 -C /usr/local/bin/ \
         etcd-${ETCD_VERSION}-linux-${TARGETARCH}/etcd \
         etcd-${ETCD_VERSION}-linux-${TARGETARCH}/etcdctl \
    && curl -fsSL "https://github.com/fffonion/lua-resty-openssl/archive/refs/heads/${LUA_RESTY_OPENSSL_VERSION}.tar.gz" \
       | tar xz -C /tmp \
    && mv /tmp/lua-resty-openssl-${LUA_RESTY_OPENSSL_VERSION}/lib/resty /usr/local/lua-resty-openssl

###############################################
# 阶段 2: 最终镜像（基于 Apisix）
###############################################
FROM apache/apisix:3.16.0-debian

USER root

# 安装 runit（进程管理）、openssl（生成占位证书）与 busybox
RUN apt-get update \
    && apt-get install -y --no-install-recommends runit openssl busybox-static \
    && busybox --install -s /usr/local/bin \
    && rm -rf /var/lib/apt/lists/*

# 拷贝二进制文件和配置
COPY --from=builder /usr/local/bin/etcd /usr/local/bin/etcdctl /usr/local/bin/isrvd /usr/local/bin/
COPY --chmod=755 build/docker-apisix/entrypoint.sh /entrypoint.sh
COPY build/docker-apisix/apisix.yaml build/docker-apisix/isrvd.yml /etc/defaults/
COPY build/docker-apisix/runit /etc/sv

# 复制 lua openssl 插件
COPY --from=builder /usr/local/lua-resty-openssl /usr/local/openresty/lualib/resty/

# 设置环境变量
ENV CONFIG_PATH=/data/conf/isrvd.yml

# 暴露端口
# 8080: isrvd Web UI
# 9180: Apisix Admin API
# 9080: Apisix 代理端口
# 9443: Apisix 代理端口 (HTTPS)
EXPOSE 8080 9180 9080 9443

# 数据卷
VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
