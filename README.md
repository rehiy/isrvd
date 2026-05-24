# iSrvd

> 名称源自 *"it is a server daemon"*，`srv` 对应 Linux 惯例目录 `/srv`，`d` 代表 daemon。**发音**：读作 **"I served"**（/ˈaɪ sɜːrvd/），谐音"爱服务"。

基于 Go + Vue 3 构建的轻量级运维面板，集成文件管理、Docker 容器编排、APISIX/Caddy 网关配置、Web 终端、GPU 监控、计划任务与 AI 助手，为个人服务器与中小型团队提供一站式管理体验。

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

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.24+ / Gin / golang-jwt |
| 前端 | Vue 3 / TypeScript / Tailwind CSS / Pinia |
| 终端 | xterm.js |
| 容器 | Docker / APISIX / Caddy |
| AI | 兼容 OpenAI API 的 LLM 接入 |

## Docker 部署

提供三个镜像版本：

| 镜像 | 说明 |
|------|------|
| `rehiy/isrvd:slim` | **默认版本**，仅含 isrvd，适合大多数场景 |
| `rehiy/isrvd:apisix` | isrvd + APISIX，集成 API 网关 |
| `rehiy/isrvd:caddy` | isrvd + Caddy，集成反向代理与 TLS 管理 |

CNB 流水线会同步推送到 CNB Docker 制品库，镜像路径为 `docker.cnb.cool/<repo-slug>:<tag>`，支持 `slim`、`caddy`、`apisix` 三个标签，其中 `slim` 同时作为 `latest`。

首次启动自动生成随机密码，通过 `docker logs isrvd` 查看。

### 创建网络

根据实际网络环境，创建 `sdnet` 网络（二选一），并将容器加入该网络。

```bash
# 本地通信
docker network create --driver=bridge sdnet
# 跨主机通信
docker network create --driver=overlay --attachable sdnet
```

### slim（默认）

仅含 isrvd 本体，体积最小，适合只需要文件管理、Docker/Swarm/Compose、计划任务等功能的场景。

```bash
docker run -d \
  --name isrvd \
  --network sdnet \
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
docker run -d \
  --name isrvd \
  --network sdnet \
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
  --network sdnet \
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

- 目录：`/usr/local/isrvd/`，包含二进制和配置文件
- 限制：无法通过容器内网访问其它容器

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
| `oidc` | OIDC 认证（issuerUrl / clientId / clientSecret / redirectUrl） |
| `agent` | AI 助手模型接入（model / baseUrl / apiKey） |
| `apisix` | APISIX Admin API 地址和密钥 |
| `caddy` | Caddy Admin API 地址 |
| `docker` | Docker 守护进程地址、容器数据目录、镜像仓库账号 |
| `marketplace` | 应用市场地址 |
| `links` | 自定义快捷链接 |
| `members` | 用户账号、家目录、模块权限 |

### 权限模块

权限基于路由进行细粒度控制，每个用户可以独立授予各模块下具体 API 路由的访问权限。

**权限格式**：`<METHOD> /api/<模块>/<路由>`（如 `GET /api/overview/status`、`POST /api/account/member`）

**前端权限判断**：使用 `actions.hasPerm('<METHOD> /api/<路由>')` 控制按钮/操作的显示

**常用模块与路由示例**：

| 模块 | 路由权限点示例 | 说明 |
|------|---------------|------|
| `overview` | `GET /api/overview/status` | 系统概览 |
| `system` | `GET /api/system/config` | 系统设置 |
| `account` | `GET /api/account/members` | 成员管理（列出） |
| `account` | `POST /api/account/member` | 成员管理（创建） |
| `account` | `PUT /api/account/member/:username` | 成员管理（更新） |
| `account` | `DELETE /api/account/member/:username` | 成员管理（删除） |
| `filer` | `GET /api/filer/list` | 文件管理（列出） |
| `filer` | `POST /api/filer/upload` | 文件管理（上传） |
| `filer` | `POST /api/filer/modify` | 文件管理（修改） |
| `shell` | `GET /api/shell` | Web 终端 |
| `agent` | `ANY /api/agent/*path` | AI 助手（LLM 代理） |
| `cron` | `GET /api/cron/jobs` | 计划任务（列出） |
| `cron` | `POST /api/cron/jobs` | 计划任务（创建） |
| `cron` | `POST /api/cron/jobs/:id/run` | 计划任务（立即执行） |
| `cron` | `GET /api/cron/jobs/:id/logs` | 计划任务（查看日志） |
| `apisix` | `GET /api/apisix/routes` | APISIX 管理（列出） |
| `docker` | `GET /api/docker/containers` | Docker 管理（列出） |
| `swarm` | `GET /api/swarm/services` | Swarm 管理（列出） |
| `compose` | `POST /api/compose/deploy` | Compose 管理（部署） |

> 留空 = 无权限；具体可用路由见各模块 API 文档

## GPU 监控

支持自动检测 NVIDIA / AMD / Intel / Apple Silicon 独立显卡，显示使用率、显存、温度、功耗、风扇转速。

| 厂商 | 首选方式 | 回退方式 |
|------|---------|---------|
| NVIDIA | go-nvml | nvidia-smi |
| AMD | sysfs | rocm-smi |
| Intel | sysfs | — |
| Apple Silicon | sysctl | — |

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
- **状态与权限**：全局状态通过 Pinia `usePortal()` 聚合访问，权限统一使用 `portal.hasPerm(moduleOrRoute)` 判断
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
