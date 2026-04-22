# isrvd

轻量级 Web 服务器管理工具，基于 Go + Vue 3 构建，提供文件管理、在线编辑、Docker/Swarm/Compose 管理、APISIX 管理和实时终端功能。

## 功能特性

| 模块 | 功能 |
|------|------|
| 系统概览 | CPU、内存、硬盘、网络、GPU 实时监控 |
| 文件管理 | 浏览、上传、下载、在线编辑、压缩解压、权限修改 |
| Web 终端 | 基于 xterm.js 的实时 Shell 交互与容器终端 |
| APISIX | 路由、Consumer、IP 白名单管理 |
| Docker | 容器、镜像、网络、卷、镜像仓库的完整管理，支持终端和实时统计 |
| Docker Swarm | 服务、节点、任务的完整管理 |
| Docker Compose | Compose 文件编辑、Docker/Swarm 一键部署与重新部署 |
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

```bash
./isrvd
```

**配置说明**

| 配置段 | 说明 |
|--------|------|
| `server` | 服务端口、JWT 密钥、代理认证头、数据目录等基础配置 |
| `agent` | AI 助手模型接入（model / baseUrl / apiKey） |
| `apisix` | APISIX Admin API 地址和密钥 |
| `docker` | Docker 守护进程地址、容器数据目录、镜像仓库账号 |
| `marketplace` | 应用市场地址 |
| `members` | 用户账号、家目录、终端权限、模块权限（主账号拥有全部权限） |

支持环境变量 `CONFIG_PATH` 指定配置文件路径（默认 `config.yml`）。

## GPU 监控

系统概览页面支持自动检测并显示 NVIDIA、AMD、Intel 独立显卡的实时指标，包括使用率、显存、温度、功耗和风扇转速。

### 支持的厂商与采集方式

| 厂商 | 首选方式 | 回退方式 | 显示指标 |
|------|---------|---------|---------|
| NVIDIA | go-nvml（直读驱动） | nvidia-smi CLI | 使用率、显存、温度、功耗、风扇 |
| AMD | sysfs（/sys/class/drm） | rocm-smi CLI | 使用率、显存、温度、功耗、风扇 |
| Intel | sysfs（/sys/class/drm） | — | 使用率、显存、温度、功耗 |

### 自动过滤

以下设备会被自动跳过，不在页面上显示：

- **虚拟显卡**：VMware、QEMU/Bochs、VirtIO、Hyper-V、VirtualBox
- **Intel 核显**：HD Graphics、UHD Graphics、Iris 系列（Intel Arc 独显保留）
- **AMD APU 核显**：无独立 VRAM 的集成显卡

### Docker 部署注意事项

容器内需要访问 GPU 设备和驱动才能采集数据：

**NVIDIA**：使用 `--gpus all` 或安装 NVIDIA Container Toolkit：

```bash
docker run -d \
  --name isrvd \
  --gpus all \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /mnt/isrvd:/data \
  rehiy/isrvd:slim
```

**AMD / Intel**：挂载 DRM 设备：

```bash
docker run -d \
  --name isrvd \
  --device /dev/dri:/dev/dri \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /mnt/isrvd:/data \
  rehiy/isrvd:slim
```

> 无 GPU 或驱动未安装的环境下，GPU 区域不会显示，不影响其他功能。

## 编译

需要 Go 1.25+ 和 Node.js 22+：

```bash
./build.sh
```

## 许可证

本项目基于 **GNU General Public License v3.0** 发布，详见 [LICENSE](LICENSE)。

### 第三方组件

本软件使用了多个第三方开源组件，各组件的授权协议详见 [NOTICE](NOTICE)。

所有第三方依赖均采用宽松型开源协议（MIT、Apache-2.0、BSD-2-Clause、BSD-3-Clause、CC-BY-4.0、OFL-1.1），与 GPL-3.0 完全兼容。
