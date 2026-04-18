###############################################
# 阶段 1: 编译 isrvd（Go 后端 + Vue 前端）
###############################################
FROM golang:1.25-bookworm AS builder-isrvd

# 安装 Node.js 用于前端构建
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs

WORKDIR /src
COPY . .

# 构建前端
RUN cd webview && npm ci && npm run build

# 构建后端（静态资源已通过 embed 嵌入）
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /src/dist/isrvd cmd/server/main.go

###############################################
# 阶段 2: 下载 etcd
###############################################
FROM debian:bookworm-slim AS builder-etcd

ARG ETCD_VERSION=v3.5.29
ARG TARGETARCH=amd64

RUN apt-get update && apt-get install -y curl tar \
    && curl -fsSL "https://github.com/etcd-io/etcd/releases/download/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-${TARGETARCH}.tar.gz" \
       | tar xz --strip-components=1 -C /usr/local/bin/ \
         etcd-${ETCD_VERSION}-linux-${TARGETARCH}/etcd \
         etcd-${ETCD_VERSION}-linux-${TARGETARCH}/etcdctl

###############################################
# 阶段 3: 最终镜像（基于 Apisix）
###############################################
FROM apache/apisix:3.16.0-debian

USER root

# 安装 supervisord
RUN apt-get update \
    && apt-get install -y --no-install-recommends supervisor \
    && rm -rf /var/lib/apt/lists/*

# 拷贝 etcd
COPY --from=builder-etcd /usr/local/bin/etcd /usr/local/bin/etcd
COPY --from=builder-etcd /usr/local/bin/etcdctl /usr/local/bin/etcdctl

# 拷贝 isrvd
COPY --from=builder-isrvd /src/dist/isrvd /usr/local/bin/isrvd

# 拷贝默认配置模板（entrypoint 启动时自动填充到 /data/conf/）
COPY build/docker/apisix.yaml /etc/defaults/apisix.yaml
COPY build/docker/isrvd.yml /etc/defaults/isrvd.yml

# 拷贝 supervisord 配置和启动脚本
COPY build/docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY build/docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# 暴露端口
# 8080: isrvd Web UI
# 9080: Apisix 代理端口
# 9180: Apisix Admin API
EXPOSE 8080 9080 9180

# 数据卷
VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
