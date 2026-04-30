###############################################
# 阶段 1: 下载 isrvd 和 etcd
###############################################
FROM alpine:3 AS builder

ARG TARGETARCH
ARG ISRVD_VERSION
ARG ETCD_VERSION=v3.5.29

RUN apk add --no-cache curl tar \
    && curl -fsSL "https://github.com/rehiy/isrvd/releases/download/${ISRVD_VERSION}/isrvd-linux-${TARGETARCH}.tar.gz" \
       | tar xz -C /usr/local/bin/ isrvd-linux-${TARGETARCH} \
    && mv /usr/local/bin/isrvd-linux-${TARGETARCH} /usr/local/bin/isrvd \
    && curl -fsSL "https://github.com/etcd-io/etcd/releases/download/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-${TARGETARCH}.tar.gz" \
       | tar xz --strip-components=1 -C /usr/local/bin/ \
         etcd-${ETCD_VERSION}-linux-${TARGETARCH}/etcd \
         etcd-${ETCD_VERSION}-linux-${TARGETARCH}/etcdctl

###############################################
# 阶段 2: 最终镜像（基于 Apisix）
###############################################
FROM apache/apisix:3.16.0-debian

USER root

# 安装 runit、openssl 和 busybox
RUN apt-get update \
    && apt-get install -y --no-install-recommends runit openssl busybox-static \
    && rm -rf /var/lib/apt/lists/*

# 拷贝二进制文件和配置
COPY --from=builder /usr/local/bin/etcd /usr/local/bin/etcdctl /usr/local/bin/isrvd /usr/local/bin/
COPY --chmod=755 build/docker/entrypoint.sh /entrypoint.sh
COPY build/docker/apisix.yaml build/docker/isrvd.yml /etc/defaults/
COPY build/docker/runit /etc/sv

# 暴露端口
# 8080: isrvd Web UI
# 9180: Apisix Admin API
# 9080: Apisix 代理端口
# 9443: Apisix 代理端口 (HTTPS)
EXPOSE 8080 9180 9080 9443

# 数据卷
VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
