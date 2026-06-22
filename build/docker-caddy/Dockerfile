###############################################
# 阶段 1: 下载 isrvd / caddy
###############################################
FROM alpine:3 AS builder

ARG TARGETARCH
ARG ISRVD_VERSION
ARG ISRVD_DOWNLOAD_BASE=https://github.com/rehiy/isrvd/releases/download
ARG CADDY_VERSION=2.8.4

RUN apk add --no-cache curl tar \
    && isrvd_download_url="${ISRVD_DOWNLOAD_BASE}/${ISRVD_VERSION}/isrvd-linux-${TARGETARCH}.tar.gz" \
    && curl -fsSL "${isrvd_download_url}" \
       | tar xz -C /usr/local/bin/ isrvd-linux-${TARGETARCH} \
    && mv /usr/local/bin/isrvd-linux-${TARGETARCH} /usr/local/bin/isrvd \
    && curl -fsSL "https://github.com/caddyserver/caddy/releases/download/v${CADDY_VERSION}/caddy_${CADDY_VERSION}_linux_${TARGETARCH}.tar.gz" \
       | tar xz -C /usr/local/bin/ caddy

###############################################
# 阶段 2: 最终镜像（isrvd + Caddy）
###############################################
FROM alpine:3

# 安装运行依赖
RUN apk add --no-cache bash ca-certificates runit

# 复制二进制文件和配置
COPY --from=builder /usr/local/bin/isrvd /usr/local/bin/caddy /usr/local/bin/
COPY --chmod=755 build/docker-caddy/entrypoint.sh /entrypoint.sh
COPY build/docker-caddy/isrvd.yml build/docker-caddy/caddy.json /etc/defaults/
COPY build/docker-caddy/runit /etc/sv

# 设置环境变量
ENV CONFIG_PATH=/data/conf/isrvd.yml

# 暴露端口
# 8080: isrvd Web UI
# 80/443: Caddy 监听端口
EXPOSE 8080 80 443

# 数据卷
VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
