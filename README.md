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

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25+ / Gin |
| 前端 | Vue 3 + TypeScript + Tailwind CSS |
| 构建 | Vite |
| 终端 | xterm.js + WebSocket |
| 认证 | JWT |
| 容器 | Docker SDK for Go |
| 监控 | gopsutil + ghw |
| 配置存储 | YAML + etcd（可选） |

## 项目结构

```text
isrvd/
├── cmd/server/          # 服务入口
├── config/              # 配置加载与保存
├── internal/
│   ├── helper/          # 辅助函数（响应、密码、WebSocket）
│   ├── registry/        # 外部服务注册（Docker、APISIX、Compose）
│   ├── server/          # HTTP handlers
│   └── service/         # 业务逻辑层
├── pkgs/
│   ├── apisix/          # APISIX Admin API 客户端
│   ├── archive/         # 压缩解压工具
│   ├── compose/         # Compose 文件解析与部署
│   ├── docker/          # Docker 客户端封装
│   ├── gpu/             # GPU 监控（NVIDIA/AMD/Intel）
│   └── swarm/           # Docker Swarm 客户端
├── webview/             # 前端 Vue 应用
│   └── src/
│       ├── component/   # 通用组件
│       ├── helper/      # 工具函数
│       ├── router/      # 路由配置
│       ├── service/     # API 服务与类型定义
│       ├── store/       # 状态管理
│       └── views/       # 页面组件
└── build/               # 构建脚本与 Dockerfile
```

## Docker 部署（推荐）

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

从 [Releases](https://github.com/rehiy/isrvd/releases) 下载压缩包：

```bash
tar xzf isrvd-linux-amd64.tar.gz
cd isrvd-linux-amd64
sudo ./systemctl/install.sh   # 安装
sudo ./systemctl/update.sh    # 更新
sudo ./systemctl/uninstall.sh # 卸载
```

配置文件：`/etc/isrvd/config.yml`

也可直接运行 `./isrvd`，通过环境变量 `CONFIG_PATH` 指定配置文件。

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
| `etcd` | etcd 连接配置（可选，启用后全局配置自动同步） |

### 配置分层存储

isrvd 支持 **YAML + etcd** 双层存储：

| 存储位置 | 配置项 | 说明 |
|----------|--------|------|
| **YAML（本地）** | `server.listenAddr`, `server.rootDirectory`, `server.debug`, `docker.host`, `docker.containerRoot`, `apisix.adminUrl`, `etcd.*` | 节点级配置，每个实例独立 |
| **etcd（全局）** | `members`, `links`, `marketplace`, `docker.registries`, `agent.*`, `server.jwtSecret`, `server.proxyHeaderName`, `apisix.adminKey` | 多实例共享，修改后自动热同步 |

未配置 `etcd` 段时，所有配置完全走 YAML，行为与之前一致。

### etcd 配置示例

```yaml
etcd:
  endpoints:
    - http://127.0.0.1:2379
  prefix: /isrvd          # 默认 /isrvd
  username: ""            # 可选
  password: ""            # 可选
  tls:                    # 可选
    certFile: ""
    keyFile: ""
    caFile: ""
```

All-in-One 镜像（`rehiy/isrvd:latest`）已内置 etcd，可直接使用本地 `http://127.0.0.1:2379`。slim 镜像需外接 etcd。

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

```text
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
- WebSocket 连接需经过认证链路
- 支持代理认证头（`proxyHeaderName`），可与反向代理集成
- etcd 传输支持 TLS 加密（可选配置 `etcd.tls`）

## 许可证

本项目基于 **GNU General Public License v3.0** 发布，详见 [LICENSE](LICENSE)。

第三方组件协议详见 [NOTICE](NOTICE)。
