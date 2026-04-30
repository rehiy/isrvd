###############################################
# 最终镜像（alpine）
###############################################
FROM alpine:3

ARG TARGETARCH
ARG ISRVD_VERSION

# 安装基础工具，下载 isrvd，清理 curl
RUN apk add --no-cache ca-certificates curl \
    && curl -fsSL "https://github.com/rehiy/isrvd/releases/download/${ISRVD_VERSION}/isrvd-linux-${TARGETARCH}.tar.gz" \
       | tar xz -C /usr/local/bin/ isrvd-linux-${TARGETARCH} \
    && mv /usr/local/bin/isrvd-linux-${TARGETARCH} /usr/local/bin/isrvd \
    && apk del curl

# 拷贝脚本文件和配置
COPY --chmod=755 build/docker-slim/entrypoint.sh /entrypoint.sh
COPY build/docker-slim/isrvd.yml /etc/defaults/isrvd.yml

# 暴露端口
# 8080: isrvd Web UI
EXPOSE 8080

# 数据卷
VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
