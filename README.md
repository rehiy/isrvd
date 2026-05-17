# isrvd

> 名称源自 *"it is a server daemon"*，`srv` 对应 Linux 惯例目录 `/srv`，`d` 代表 daemon。

轻量级 Web 服务器管理工具，基于 Go + Vue 3 构建，提供文件管理、Docker/Swarm/Compose 管理、APISIX 管理和实时终端等功能。

## 功能特性

| 模块 | 功能 |
|------|------|
| 系统概览 | CPU、内存、磁盘、网络、GPU 实时监控 |
| 文件管理 | 浏览、上传、下载、编辑、压缩解压、权限修改 |
| Web 终端 | 基于 xterm.js 的 Shell 与容器终端 |
| AI 助手 | 内置 Agent，支持自然语言操作 |
| 计划任务 | 定时任务调度，支持 Shell/BAT/PowerShell/Docker 执行模式 |
| APISIX | 路由、Consumer、上游、SSL、插件配置、白名单 |
| Caddy | 路由、TLS 证书、全局选项管理 |
| Docker | 容器、镜像、网络、卷、镜像仓库管理 |
| Swarm | 集群、节点、服务、任务管理 |
| Compose | 文件编辑、Docker/Swarm 部署与重部署 |
| 成员管理 | 多用户、家目录隔离、模块权限控制 |
| 系统管理 | 配置管理、操作审计日志 |
| 移动端 | 响应式布局，适配移动设备 |

## Docker 部署

提供三个镜像版本：

| 镜像 | 说明 |
|------|------|
| `rehiy/isrvd:slim` | **默认版本**，仅含 isrvd，适合大多数场景 |
| `rehiy/isrvd:apisix` | isrvd + APISIX，集成 API 网关 |
| `rehiy/isrvd:caddy` | isrvd + Caddy，集成反向代理与 TLS 管理 |

CNB 流水线会同步推送到 CNB Docker 制品库，镜像路径为 `docker.cnb.cool/<repo-slug>:<tag>`，支持 `slim`、`caddy`、`apisix` 三个标签，其中 `slim` 同时作为 `latest`。

首次启动自动生成随机密码，通过 `docker logs isrvd` 查看。

### slim（默认）

仅含 isrvd 本体，体积最小，适合只需要文件管理、Docker/Swarm/Compose、计划任务等功能的场景。

```bash
docker run -d \
  --name isrvd \
  -p 8080:8080 \
  -v /srv/data:/data \
  -v /var/run/docker.sock:/var/run/docker.sock \
  rehiy/isrvd:slim
```

| 端口 | 服务 | 说明 |
|------|------|------|
| 8080 | isrvd | Web 管理界面 |

### apisix（集成 API 网关）

isrvd + APISIX，适合已使用 APISIX 作为 API 网关的场景。

```bash
docker network create srvdnet
docker run -d \
  --name isrvd \
  --network srvdnet \
  -p 8080:8080 \
  -p 80:9080 \
  -p 443:9443 \
  -v /srv/data:/data \
  -v /var/run/docker.sock:/var/run/docker.sock \
  rehiy/isrvd:apisix
```

| 端口 | 服务 | 说明 |
|------|------|------|
| 8080 | isrvd | Web 管理界面 |
| 9080 | APISIX | HTTP 代理端口 |
| 9443 | APISIX | HTTPS 代理端口 |

### caddy（集成反向代理）

isrvd + Caddy，适合需要反向代理、自动 HTTPS（ACME）或统一网关管理的场景。

```bash
docker run -d \
  --name isrvd \
  -p 8080:8080 \
  -p 80:80 \
  -p 443:443 \
  -v /srv/data:/data \
  -v /var/run/docker.sock:/var/run/docker.sock \
  rehiy/isrvd:caddy
```

| 端口 | 服务 | 说明 |
|------|------|------|
| 8080 | isrvd | Web 管理界面 |
| 80 | Caddy | HTTP 代理端口 |
| 443 | Caddy | HTTPS 代理端口 |

Caddy 默认配置仅监听 `:80`，关闭了自动 HTTPS 跳转，首次启动即可正常访问。如需 HTTPS，在「Caddy → 全局选项」中开启，或直接编辑「原始配置」。

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
      - /srv/data:/data
      - /var/run/docker.sock:/var/run/docker.sock
```

> **注意**：请始终挂载整个 `/data` 目录，避免容器重建时数据丢失。

## 二进制部署

安装目录：`/usr/local/isrvd/`（包含二进制和配置文件）

```bash
# 一键安装
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) install

# 更新/卸载/仅下载
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) update
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) uninstall
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) download
```

也可直接运行 `./isrvd`，通过 `CONFIG_PATH` 指定配置位置：

```bash
# 默认读取 ./config.yml
./isrvd

# 本地 YAML
CONFIG_PATH=/data/conf/isrvd.yml ./isrvd

# etcd（value 仍为 config.yml 同款 YAML）
etcdctl put /isrvd/config "$(cat /data/conf/isrvd.yml)"
CONFIG_PATH="etcd://user:pass@127.0.0.1:2379/isrvd/config?scheme=http&timeout=5s" ./isrvd

# etcd key 不存在时，用 fallback YAML 初始化并写入 etcd
CONFIG_PATH="etcd://127.0.0.1:2379/isrvd/config?fallback=/data/conf/isrvd.yml" ./isrvd

# etcd 完整配置示例
# etcd://user:pass@host1:2379,host2:2379/key?scheme=http&timeout=5s&fallback=/path/config.yml
```

**etcd** 认证可省略或用 `ETCD_USERNAME` / `ETCD_PASSWORD` 补充，配置变更仅记录重启提示，不自动热更新。

## 本地开发

### 环境要求

- **Go**：用于后端服务与命令行构建
- **Node.js / npm**：用于 `webview` 前端开发与构建
- **Docker**：可选，用于 Docker、Swarm、Compose 相关功能调试

### 启动开发环境

```bash
./develop.sh
```

开发脚本会自动：

- **后端**：复制 `config.yml` 为 `.local.yml`（如不存在），并通过 `CONFIG_PATH=.local.yml go run cmd/server/main.go` 启动
- **前端**：进入 `webview`，安装依赖并执行 `npm run dev`
- **端口清理**：启动前尝试释放 `8080` 和 `3000` 端口

Windows 环境可使用：

```bat
develop.bat
```

### 构建与校验

```bash
# 完整分发构建
./build.sh

# 后端测试
go test ./...

# 前端类型检查
cd webview && npm run lint

# 前端 import 排序检查
cd webview && python3 sort-imports.py --dry-run src
```

> 贡献代码前请优先阅读 [AGENTS.md](AGENTS.md)。该文件是当前仓库的代码规范与协作约定入口，旧版 `CODE_STYLE` 不再作为规范来源。

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

权限基于路由进行细粒度控制，每个用户可以独立授予各模块下具体 API 路由的访问权限。

**权限格式**：`POST /api/<模块>/<路由>`（具体 HTTP 方法与路径）

**前端权限判断**：使用 `actions.hasPerm('POST /api/<路由>')` 控制按钮/操作的显示

**常用模块与路由示例**：

| 模块 | 路由权限点示例 | 说明 |
|------|---------------|------|
| `overview` | `POST /api/overview/stats` | 系统概览 |
| `system` | `POST /api/system/config` | 系统设置 |
| `account` | `POST /api/members` | 成员管理 |
| `filer` | `GET /api/filer/list` | 文件管理（列出） |
| `filer` | `POST /api/filer/upload` | 文件管理（上传） |
| `filer` | `POST /api/filer/modify` | 文件管理（修改） |
| `shell` | `WS /api/shell` | Web 终端 |
| `agent` | `POST /api/agent/chat` | AI 助手 |
| `cron` | `POST /api/cron/jobs` | 计划任务管理 |
| `cron` | `POST /api/cron/jobs/:id/run` | 计划任务（立即执行） |
| `cron` | `GET /api/cron/jobs/:id/logs` | 计划任务（查看日志） |
| `apisix` | `POST /api/apisix/routes` | APISIX 管理 |
| `docker` | `POST /api/docker/containers` | Docker 管理 |
| `swarm` | `POST /api/swarm/services` | Swarm 管理 |
| `compose` | `POST /api/compose/deploy` | Compose 管理 |

> 留空 = 无权限；具体可用路由见各模块 API 文档

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
docker run -d --gpus all rehiy/isrvd:slim
```

**AMD / Intel**：

```bash
docker run -d --device /dev/dri:/dev/dri rehiy/isrvd:slim
```

## 架构设计

### 分层架构

```text
config → registry → pkgs → service → server
```

- **config**：配置加载
- **registry**：外部服务初始化
- **pkgs**：各模块客户端（Docker / Swarm / APISIX / Caddy / GPU / Compose / Archive）
- **service**：业务逻辑与类型转换
- **server**：HTTP handlers

### 设计原则

- **高内聚**：同一领域功能聚合在同一包
- **低耦合**：层间通过接口解耦
- **单一职责**：Handler 只管 HTTP，Service 只管业务

### 开发规范

- **后端分层**：`pkgs` 保持原生客户端能力，`service` 负责业务组合与类型转换，`server` 只处理 HTTP 入出口
- **前端结构**：`webview/src/service/types` 按域拆分类型，页面复用统一卡片、表格、移动端双视图和操作按钮语义色
- **状态与权限**：全局状态通过 Provide/Inject 注入，权限统一使用 `hasPerm(module, write?)` 判断
- **安全基线**：敏感字段不返回明文，文件路径与解压路径必须校验，WebSocket 必须经过认证链路
- **完整规范**：详见 [AGENTS.md](AGENTS.md)

## 安全特性

- JWT 认证，敏感字段（密钥、密码）不返回前端
- 文件路径校验，防止目录遍历攻击
- 解压校验路径，防止 Zip Slip 攻击
- WebSocket 连接需经过认证中间件
- 基于路由的细粒度权限控制，路由访问级别支持 `0` 需权限、`1` 需登录、`-1` 匿名
- 操作审计日志，路由审计级别支持 `0` 按 Method、`-1` 忽略、`1` 强制记录，默认记录非 GET 请求与 WebSocket 连接
- 支持代理认证头（`proxyHeaderName`）

## 许可证

本项目基于 **Apache License 2.0** 发布，详见 [LICENSE](LICENSE)。

第三方组件协议详见 [NOTICE](NOTICE)。
