# isrvd

轻量级 Web 服务器管理工具，基于 Go + Vue 3 构建，提供文件管理、Docker/Swarm/Compose 管理、APISIX 管理和实时终端等功能。

## 功能特性

| 模块 | 功能 |
|------|------|
| 系统概览 | CPU、内存、磁盘、网络、GPU 实时监控 |
| 文件管理 | 浏览、上传、下载、编辑、压缩解压、权限修改 |
| Web 终端 | 基于 xterm.js 的 Shell 与容器终端 |
| AI 助手 | 内置 Agent，支持自然语言操作 |
| APISIX | 路由、Consumer、上游、SSL、插件配置 |
| Docker | 容器、镜像、网络、卷、镜像仓库管理 |
| Swarm | 集群、节点、服务、任务管理 |
| Compose | 文件编辑、Docker/Swarm 部署与重部署 |
| 成员管理 | 多用户、家目录隔离、模块权限控制 |
| 移动端 | 响应式布局，适配移动设备 |

## Docker 部署

提供两个镜像版本：

| 镜像 | 说明 |
|------|------|
| `rehiy/isrvd:latest` | All-in-One，集成 APISIX + etcd + isrvd |
| `rehiy/isrvd:slim` | 仅 isrvd，适合不需要 APISIX 的场景 |

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

| 端口 | 服务 | 说明 |
|------|------|------|
| 8080 | isrvd | Web 管理界面 |
| 9080 | APISIX | HTTP 代理端口 |

### slim（仅 isrvd）

```bash
docker run -d \
  --name isrvd \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /mnt/isrvd:/data \
  rehiy/isrvd:slim
```

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

> **注意**：请始终挂载整个 `/data` 目录，避免容器重建时数据丢失。

## 二进制部署

从 [Releases](https://github.com/rehiy/isrvd/releases) 下载二进制文件，编辑 `config.yml` 后运行：

```bash
./isrvd
```

支持环境变量 `CONFIG_PATH` 指定配置文件路径（默认 `config.yml`）。

### 配置说明

| 配置段 | 说明 |
|--------|------|
| `server` | 端口、JWT 密钥、代理认证头、数据目录 |
| `agent` | AI 助手模型接入（model / baseUrl / apiKey） |
| `apisix` | APISIX Admin API 地址和密钥 |
| `docker` | Docker 守护进程地址、容器数据目录、镜像仓库账号 |
| `marketplace` | 应用市场地址 |
| `links` | 自定义快捷链接 |
| `members` | 用户账号、家目录、模块权限 |

### 权限模块

| 模块 | 权限值 | 说明 |
|------|--------|------|
| `system` | `r` / `rw` | 系统设置 |
| `filer` | `r` / `rw` | 文件管理 |
| `shell` | `r` / `rw` | Web 终端 |
| `agent` | `r` / `rw` | AI 助手 |
| `apisix` | `r` / `rw` | APISIX 管理 |
| `docker` | `r` / `rw` | Docker 管理 |
| `swarm` | `r` / `rw` | Swarm 管理 |
| `compose` | `r` / `rw` | Compose 管理 |

> `r` = 只读，`rw` = 读写，留空 = 无权限

## GPU 监控

支持自动检测 NVIDIA / AMD / Intel 独立显卡，显示使用率、显存、温度、功耗、风扇转速。

| 厂商 | 首选方式 | 回退方式 |
|------|---------|---------|
| NVIDIA | go-nvml | nvidia-smi |
| AMD | sysfs | rocm-smi |
| Intel | sysfs | — |

自动过滤虚拟显卡与核显（Intel Arc 独显保留）。

### Docker 部署注意事项

**NVIDIA**：

```bash
docker run -d --gpus all -p 8080:8080 -v /var/run/docker.sock:/var/run/docker.sock -v /mnt/isrvd:/data rehiy/isrvd:slim
```

**AMD / Intel**：

```bash
docker run -d --device /dev/dri:/dev/dri -p 8080:8080 -v /var/run/docker.sock:/var/run/docker.sock -v /mnt/isrvd:/data rehiy/isrvd:slim
```

## 架构设计

### 分层架构

```
config → registry → pkgs → service → server
```

- **config**：配置加载
- **registry**：外部服务初始化
- **pkgs**：各模块客户端（Docker / Swarm / APISIX / GPU / Compose）
- **service**：业务逻辑与类型转换
- **server**：HTTP handlers

### 设计原则

- **高内聚**：同一领域功能聚合在同一包
- **低耦合**：层间通过接口解耦
- **单一职责**：Handler 只管 HTTP，Service 只管业务

## 编译

需要 Go 1.25+ 和 Node.js 22+：

```bash
./build.sh
```

## 安全特性

- JWT 认证，敏感配置仅返回 `xxxSet` 布尔值
- 文件系统操作防目录遍历攻击
- 压缩解压防 Zip Slip 攻击
- WebSocket 连接需经过认证
- 支持代理认证头（`proxyHeaderName`）

## 许可证

本项目基于 **GNU General Public License v3.0** 发布，详见 [LICENSE](LICENSE)。

第三方组件协议详见 [NOTICE](NOTICE)。
