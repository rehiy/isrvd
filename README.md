# Isrvd

轻量级 Web 服务器管理工具，基于 Go + Vue 3 构建，提供文件管理、在线编辑、Docker/Swarm 管理、APISIX 管理和实时终端功能。

## 特性

- **系统概览** - CPU、内存、硬盘、网络实时监控
- **文件管理** - 浏览、上传、下载、在线编辑、压缩解压、权限修改
- **Docker 管理** - 容器、镜像、网络、卷的完整管理，支持终端和实时统计
- **Docker Swarm** - 服务、节点、任务的完整管理
- **APISIX 管理** - 路由、Consumer、Upstream、IP 白名单管理
- **Web 终端** - xterm.js 实时 Shell 交互
- **多用户** - 用户隔离、独立家目录、权限控制
- **移动端适配** - 响应式布局

## Docker 部署

镜像将 **APISIX + etcd + isrvd** 打包为 All-in-One 容器，由 supervisord 管理所有服务。

```bash
docker run -d \
  --name isrvd \
  -p 8080:8080 \
  -p 9080:9080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /mnt/isrvd:/data \
  rehiy/isrvd:latest
```

首次启动会自动生成随机密码，通过 `docker logs isrvd` 查看。

### Docker Compose

```yaml
services:
  isrvd:
    image: rehiy/isrvd:latest
    container_name: isrvd
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "9080:9080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /mnt/isrvd:/data
```

### 端口说明

| 端口 | 服务 | 说明 |
|------|------|------|
| 8080 | isrvd | Web 管理界面 |
| 9080 | APISIX | HTTP 代理端口 |
| 9180 | APISIX | Admin API（默认不对外暴露） |

### 数据与配置

容器使用 `/data` 作为数据根目录，包含以下子目录：

| 路径 | 说明 |
|------|------|
| `/data/conf/` | 配置文件（apisix.yaml、isrvd.yml） |
| `/data/etcd/` | etcd 数据 |
| `/data/storage/` | 文件管理根目录 |
| `/data/container/` | 容器数据目录 |

首次启动后配置文件自动生成到 `/data/conf/`，编辑后重启容器即可生效。也可预先准备配置文件放到本地 `data/conf/` 目录再挂载。

> **注意**：请始终挂载整个 `/data` 目录，不要单独挂载子目录，否则未挂载的数据会在容器重建时丢失。

## 二进制部署

从 [Releases](https://github.com/rehiy/isrvd/releases) 下载对应平台二进制文件，编辑 `config.yml` 后运行：

```yaml
server:
  listenAddr: ":8080"
  jwtSecret: your-secret-key
  rootDirectory: "."

members:
  - username: admin
    password: admin123
    homeDirectory: public
    allowTerminal: true
```

```bash
./isrvd
```

支持环境变量覆盖：`DEBUG`、`LISTEN_ADDR`、`ROOT_DIRECTORY`、`JWT_SECRET`、`PROXY_HEADER_NAME`

## 编译

需要 Go 1.21+ 和 Node.js 16+：

```bash
./build.sh
```

## 许可证

GPL-3.0
