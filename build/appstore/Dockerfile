# 构建阶段：运行 build.py 生成 index.json 和 storage/
FROM python:3.12-slim AS builder

WORKDIR /build

COPY build/appstore/build.py ./
RUN pip install --no-cache-dir pyyaml && python build.py

# 运行阶段：nginx 静态文件服务
FROM nginx:alpine

COPY build/appstore/index.html /usr/share/nginx/html/index.html
COPY --from=builder /build/index.json /usr/share/nginx/html/index.json
COPY --from=builder /build/storage /usr/share/nginx/html/storage
