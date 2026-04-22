# isrvd

轻量级 Web 服务器管理工具，基于 Go + Vue 3 构建，提供文件管理、在线编辑、Docker/Swarm/Compose 管理、APISIX 管理和实时终端功能。

## 功能特性

| 模块 | 功能 |
|------|------|
| 系统概览 | CPU、内存、硬盘、网络实时监控 |
| 文件管理 | 浏览、上传、下载、在线编辑、压缩解压、权限修改 |
| Docker | 容器、镜像、网络、卷的完整管理，支持终端和实时统计 |
| Docker Swarm | 服务、节点、任务的完整管理 |
| Docker Compose | Compose 文件编辑、Docker/Swarm 一键部署与重新部署 |
| APISIX | 路由、Consumer、Upstream、Plugin Config、IP 白名单管理 |
| Web 终端 | 基于 xterm.js 的实时 Shell 交互与容器终端 |
| 成员管理 | 多用户支持，用户隔离、独立家目录、权限控制 |
| AI 助手 | 内置 Page-Agent，支持自然语言操作与页面感知 |
| 移动端 | 响应式布局，全面适配移动设备 |

## Docker 部署（推荐）

提供两个镜像版本：

| 镜像 | 说明 |
|------|------|
| `rehiy/isrvd:latest` | All-in-One，集成 **APISIX + etcd + isrvd**，由 supervisord 统一管理 |
| `rehiy/isrvd:slim` | 精简版，仅包含 **isrvd**，适合不需要 APISIX 的场景 |

首次启动自动生成随机密码，通过 `docker logs isrvd` 查看。

### latest（All-in-One）

```bash
docker run -d \
  --name isrvd \
  -p 8080:8080 \
  -p 9080:9080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /mnt/isrvd:/data \
  rehiy/isrvd:latest
```

**端口说明**

| 端口 | 服务 | 说明 |
|------|------|------|
| 8080 | isrvd | Web 管理界面 |
| 9080 | APISIX | HTTP 代理端口 |
| 9180 | APISIX | Admin API（默认不对外暴露） |

**数据目录**

| 路径 | 说明 |
|------|------|
| `/data/conf/` | 配置文件（apisix.yaml、isrvd.yml） |
| `/data/etcd/` | etcd 数据 |
| `/data/container/` | 容器数据目录 |
| `/data/{member}/` | 用户数据目录 |

### slim（仅 isrvd）

```bash
docker run -d \
  --name isrvd \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /mnt/isrvd:/data \
  rehiy/isrvd:slim
```

**端口说明**

| 端口 | 服务 | 说明 |
|------|------|------|
| 8080 | isrvd | Web 管理界面 |

**数据目录**

| 路径 | 说明 |
|------|------|
| `/data/conf/` | 配置文件（isrvd.yml） |
| `/data/container/` | 容器数据目录 |
| `/data/{member}/` | 用户数据目录 |

### Docker Compose

```yaml
services:
  isrvd:
    image: rehiy/isrvd:slim
    container_name: isrvd
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /mnt/isrvd:/data
```

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

需要 Go 1.25+ 和 Node.js 16+：

```bash
./build.sh
```

## 许可证

本项目基于 **GNU General Public License v3.0** 发布，详见 [LICENSE](LICENSE)。

### 第三方组件

本软件使用了多个第三方开源组件，各组件的授权协议详见 [NOTICE](NOTICE)。

所有第三方依赖均采用宽松型开源协议（MIT、Apache-2.0、BSD-2-Clause、BSD-3-Clause、CC-BY-4.0、OFL-1.1），与 GPL-3.0 完全兼容。
