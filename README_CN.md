# iSrvd

[English](README.md) | 简体中文

> 名称源自 *"it is a server daemon"*，`srv` 对应 Linux 惯例目录 `/srv`，`d` 代表 daemon。**发音**：读作 **"I served"**（/ˈaɪ sɜːrvd/），谐音"爱服务"。

基于 Go + Vue 3 构建的轻量级运维面板，集成文件管理、Docker 容器编排、APISIX/Caddy 网关配置、Web 终端、GPU 监控、计划任务与 AI 助手，为个人服务器与中小型团队提供一站式管理体验。

## 目录

- [功能特性](#功能特性)
- [技术栈](#技术栈)
- [部署](#部署)
- [集中管理](#集中管理)
- [配置](#配置)
- [权限](#权限)
- [GPU 监控](#gpu-监控)
- [本地开发](#本地开发)
- [架构设计](#架构设计)
- [安全特性](#安全特性)
- [许可证](#许可证)

## 功能特性

| 模块 | 功能 |
| ------ | ------ |
| 系统概览 | CPU、内存、磁盘、网络、Go 运行时与 GPU 监控，支持历史数据采集、服务可用性探测和在线升级 |
| 文件管理 | 浏览、上传、下载、编辑、创建/删除目录、重命名、权限修改、压缩/解压 |
| Web 终端 | 基于 xterm.js 的 Shell 终端，支持容器终端接入 |
| 本机进程 | 查看主机进程列表（CPU、内存、命令行），终止指定进程（强制审计） |
| SSH 远程管理 | 管理主机与可复用凭据，支持密码/私钥认证、浏览器终端和 SFTP 文件管理 |
| 集中管理 | 同一个二进制按 `--mode` 作为单机、中控或受管机运行，一个控制台管理多台服务器，详见 [集中管理](#集中管理) |
| AI 助手 | 内置 Copilot，基于 CopilotKit + AG-UI 协议，通过内置 OpenAPI 目录调用后端接口，支持页面上下文、工具卡片与写操作审批，兼容 OpenAI API 的 LLM 接入 |
| 计划任务 | 定时任务调度；按运行平台提供 Shell 或 BAT/PowerShell 脚本及可执行文件任务，Docker 可用时还支持临时容器或已有容器执行 |
| APISIX | 路由、Consumer、上游(Upstream)、SSL 证书、插件配置(PluginConfig)、插件列表、访问授权管理 |
| Caddy | HTTP 服务、路由、Basic Auth、SSL 证书、全局选项管理，支持原始配置编辑 |
| Docker | 容器、镜像、网络、卷、镜像仓库管理，容器文件、实时日志、资源统计、终端接入、镜像构建/推送/拉取 |
| Swarm | 集群信息、节点、服务、任务管理，服务日志、强制更新、加入令牌管理 |
| Compose | 文件编辑、Docker Compose / Swarm Stack 部署与重部署，最多 10 条部署记录与加密配置快照，支持加载历史配置重新部署 |
| 成员管理 | 多用户、家目录隔离、路由级权限控制、API 令牌、TOTP 二次验证和 Passkey 无密码登录 |
| 系统管理 | 分组配置管理、操作审计日志、资源及应用故障告警与 Webhook 通知、OIDC 认证集成、代理认证头登录 |
| 移动端 | 响应式布局，适配移动设备 |

## 技术栈

| 层级 | 技术 |
| ------ | ------ |
| 后端 | Go 1.26+ / Gin / golang-jwt |
| 前端 | Vue 3 / TypeScript / Tailwind CSS / Pinia |
| 终端 | xterm.js |
| 容器 | Docker / APISIX / Caddy |
| AI | 兼容 OpenAI API 的 LLM 接入，前端基于 CopilotKit + AG-UI 协议 |

## 部署

三种方式可选：安装脚本（推荐）、Docker 镜像、二进制安装包。安装脚本面向使用 systemd 的 Linux，需以 root 用户执行。

### 镜像版本

| 镜像 | 说明 |
| ------ | ------ |
| `rehiy/isrvd:slim` | **默认版本**，仅含 isrvd，适合大多数场景 |
| `rehiy/isrvd:apisix` | isrvd + APISIX，集成 API 网关 |
| `rehiy/isrvd:caddy` | isrvd + Caddy，集成反向代理与 TLS 管理 |

CNB 流水线会同步推送到 CNB Docker 制品库，镜像路径为 `docker.cnb.cool/<repo-slug>:<tag>`，支持 `slim`、`caddy`、`apisix` 三个标签，其中 `slim` 同时作为 `latest`。
国内 Docker 部署可将下方示例中的 `rehiy/isrvd:<tag>` 替换为 `docker.cnb.cool/rehiy/isrvd:<tag>`。

Docker 版默认管理员账号为 `admin` / `admin`，首次登录成功后会自动跳转至修改密码页面。

### 脚本安装（推荐）

安装脚本会自动安装 Docker、初始化单节点 Swarm、创建可挂载的 overlay 网络 `sdnet`，并根据参数启动对应的一体化镜像：

```bash
# slim / caddy / apisix 三选一
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) install --docker
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) install --caddy
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) install --apisix
```

Docker 版统一使用容器名 `isrvd`，数据保存在 `/srv/data`；`update` 会按当前镜像类型重建容器，`uninstall` 删除容器但保留数据目录。

### 准备网络

使用安装脚本时无需手动操作。手动部署 Docker 版时，推荐先初始化 Swarm，再创建可挂载的 overlay 网络：

```bash
docker swarm init
docker network create --driver=overlay --attachable sdnet
```

### 运行容器

| 镜像 | 端口映射 | 说明 |
| ------ | ---------- | ------ |
| `slim` | 8080 → 8080 | isrvd Web 管理界面 |
| `apisix` | 8080 → 8080、80 → 9080、443 → 9443 | isrvd 界面；APISIX HTTP / HTTPS 代理 |
| `caddy` | 8080 → 8080、80 → 80、443 → 443 | isrvd 界面；Caddy HTTP / HTTPS 代理 |

#### slim（默认）

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

#### apisix（集成 API 网关）

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

#### caddy（集成反向代理）

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

Caddy 默认 HTTP 服务监听 `:80` 和 `:443`，但禁用了自动 HTTPS 及重定向。如需 HTTPS，可在「Caddy → 服务」中编辑对应服务的 HTTPS 行为，或直接编辑「原始配置」。

### Docker Compose

```yaml
services:
  isrvd:
    image: rehiy/isrvd:slim
    container_name: isrvd
    restart: unless-stopped
    networks:
      - sdnet
    ports:
      - "8080:8080"
    volumes:
      - /srv/data:/data
      - /var/run/docker.sock:/var/run/docker.sock

networks:
  sdnet:
    external: true
```

> **注意**：
>
> - 请先创建 `sdnet` 网络（见上方「准备网络」章节），Compose 中通过 `external: true` 引用已有网络
> - 请始终挂载整个 `/data` 目录，避免容器重建时数据丢失

### 二进制部署

- 目录：`/usr/local/isrvd/`，包含二进制和配置文件
- 限制：无法通过容器内网访问其它容器

```bash
# 一键安装（默认自动按 IP 选择 CNB/GitHub 源，可用 --cn / --global 手动指定）
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) install
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) install --cn

# 更新/卸载/仅下载
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) update
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) update --global
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) uninstall
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd.sh) download
```

配置文件位置通过 `CONFIG_PATH` 指定，详见 [配置](#配置)。

## 集中管理

同一个二进制，用 `--mode`（或环境变量 `ISRVD_MODE`）选择角色，一个控制台管理多台服务器：

| 模式 | 作用 |
| ------ | ------ |
| `server`（默认） | 单机，与以往完全一致 |
| `center` | 中控：提供 Web 界面、账号与权限，接收受管机接入，并把请求按节点转发 |
| `agent` | 受管机：不开放任何对外端口，主动连接中控，中控转发来的请求在本机执行 |

```text
浏览器 ──> center ──┬── 页面、账号、权限、审计 ──> 本进程内的 isrvd
                    └── /n/<节点ID>/api/… ──隧道──> agent（本进程内的 isrvd）
                                                  ▲ 受管机主动出站，中控从不主动外连
```

- **无需入站端口**：受管机只需要能访问中控，位于 NAT 或防火墙后也可以接入；断线后按指数退避自动重连
- **切换节点即切换视角**：创始人在页面头部选择节点后，Docker、Swarm、Compose、文件、本机进程、终端、计划任务、APISIX、Caddy 和系统概览都作用于该节点，包括实时日志、终端和大文件上传下载
- **账号与配置仍在中控**：成员、权限、系统配置、AI 助手和 SSH 远程管理始终由中控处理，节点上的操作使用中控的登录态
- **按需出现**：节点切换器和侧边栏「节点管理」只在 `center` 模式且当前用户是创始人时显示，单机用户看不到任何变化

### 接入节点

1. 启动中控：`isrvd --mode center`，沿用原来的配置文件与 `listenAddr`
2. 在「节点管理」页点击「接入节点」生成一次性注册码，界面会给出可直接复制的受管机启动命令；未使用的注册码在列表里显示为「待接入」，可随时撤销
3. 在受管机启动：`isrvd --mode agent --center-url https://center.example.com --enroll-code <注册码>`
4. 在同一页面审批该节点；勾选「自动通过审批」生成的注册码可跳过这一步，节点随即显示「在线」

首次注册后节点凭据加密保存在受管机本地，重启不需要再带注册码。

### 容器部署

镜像入口不带参数，用环境变量选择模式，不需要改镜像或配置文件：

```bash
# 中控：与单机部署相同，多一个环境变量，8080 为对外入口
docker run -d --name isrvd-center --network sdnet -p 8080:8080 \
  -e ISRVD_MODE=center \
  -v /srv/data:/data -v /var/run/docker.sock:/var/run/docker.sock \
  rehiy/isrvd:slim

# 受管机：不监听对外端口，不需要 -p
docker run -d --name isrvd-agent --network sdnet \
  -e ISRVD_MODE=agent \
  -e ISRVD_CENTER_URL=https://center.example.com \
  -e ISRVD_ENROLL_CODE=<注册码> \
  -v /srv/data:/data -v /var/run/docker.sock:/var/run/docker.sock \
  rehiy/isrvd:slim
```

> - 受管机的节点凭据保存在 `/data` 下，请保持挂载，否则重建容器后需要新的注册码重新接入
> - 受管机的 `isrvd.yml` 里必须有创始人成员（镜像默认的 `admin` 满足），且不能启用 `tha`
> - 受管机配置里的 `listenAddr` 在此模式下不生效；`jwtSecret` 用于加密节点凭据，建议改掉镜像默认值，且之后不要再变更

### systemd 部署

安装脚本生成的是单机服务。作为中控或受管机运行时，用 drop-in 追加环境变量，无需修改脚本生成的单元文件（脚本升级后仍然有效）：

```bash
systemctl edit isrvd
# 在编辑器中写入（受管机示例；中控只需 ISRVD_MODE=center）：
#   [Service]
#   Environment="ISRVD_MODE=agent"
#   Environment="ISRVD_CENTER_URL=https://center.example.com"
#   Environment="ISRVD_ENROLL_CODE=<注册码>"
systemctl restart isrvd
```

### 安全与边界

- 只有创始人可以管理和操作节点；节点上的请求以受管机本地的创始人身份执行，不能把节点权限下放给普通成员
- 注册码一次性使用、默认 1 小时过期；节点令牌只保存哈希，注册码短期有效、加密落盘，待接入期间可在列表中查看接入命令；节点可随时吊销，吊销后立即断开
- 节点管理操作、节点上的写操作与终端会话都会记入中控的审计页面（操作人为中控登录用户），受管机本机的审计页面里则记录为其本地创始人
- 中控为单实例设计，重启时节点短暂断开后自动重连；受管机与中控的版本需一致
- 完整的参数表、接口与行为说明见 [docs/multi-node.md](docs/multi-node.md)，接口字段见 [受管节点](docs/references/node/nodes.md) 与 [注册码](docs/references/node/codes.md)

## 配置

### 配置来源

配置位置由 `CONFIG_PATH` 指定，支持本地 YAML 与 etcd（value 仍为 config.yml 同款 YAML）：

```bash
# 默认读取 ./config.yml
./isrvd

# 本地 YAML
CONFIG_PATH=/data/conf/isrvd.yml ./isrvd

# etcd
etcdctl put /isrvd/config "$(cat /data/conf/isrvd.yml)"
CONFIG_PATH="etcd://user:pass@127.0.0.1:2379/isrvd/config?scheme=http&timeout=5s" ./isrvd

# etcd key 不存在时，用 fallback YAML 初始化并写入 etcd
CONFIG_PATH="etcd://127.0.0.1:2379/isrvd/config?fallback=/data/conf/isrvd.yml" ./isrvd

# etcd 完整配置示例
# etcd://user:pass@host1:2379,host2:2379/key?scheme=http&timeout=5s&fallback=/path/config.yml
```

**etcd** 认证可省略，也可用 `ETCD_USERNAME` / `ETCD_PASSWORD` 补充或覆盖 URI 中的认证信息。etcd key 发生 PUT 变更时，isrvd 会重载配置、注册中心和业务服务。计划任务、SSH 主机与凭据等业务数据同样存入 etcd（key 为 `<配置 key>/cron.yml` 等），首次启动时自动迁移 `rootDirectory` 下的同名文件；审计与监控日志仍写本地。迁移后本地文件不再更新，回退到不支持该特性的旧版本会丢失升级期间对计划任务与 SSH 配置的修改。通过系统配置 API 保存的本地 YAML 也会立即触发重载；若直接在磁盘上修改 YAML，则需发送 `SIGHUP` 或重启进程。

### 配置项

各配置段的含义见 [配置段说明](docs/references/system/config.md#配置段说明)。

## 权限

权限基于路由进行细粒度控制。默认的 `AccessPerm` 路由要求成员持有对应的完整路由权限，Founder 不受此限制；`AccessAuth` 路由只要登录即可访问，`AccessAnon` 路由允许匿名访问。

**权限格式**：`<METHOD> /api/<模块>/<路由>`（如 `GET /api/docker/containers`、`POST /api/compose/docker`）

**前端权限判断**：使用 `portal.hasPerm('<METHOD> /api/<路由>')` 控制按钮/操作的显示

> 留空 = 无 `AccessPerm` 路由权限；具体可用路由可在登录后通过 `GET /api/account/routes` 获取。
>
> 集中管理的节点接口（`/api/node/*`、`/n/<节点ID>/…`）由中控网关提供，不在路由表中，**仅创始人可用**，无法授予普通成员。

权限点完整清单见 [docs/permissions.md](docs/permissions.md)。

## GPU 监控

支持自动检测 NVIDIA / AMD / Intel / Apple Silicon 独立显卡，显示使用率、显存、温度、功耗、风扇转速。

检测方式、采集指标与容器部署注意事项见 [docs/gpu-monitoring.md](docs/gpu-monitoring.md)。

## 本地开发

### 环境要求

- **Go**：1.26.0 或更高版本，最低版本以 `go.mod` 为准，用于后端服务与命令行构建
- **Node.js / npm**：用于 `webview` 前端开发与构建（持续集成环境使用 Node.js 24）
- **Docker**：可选，用于 Docker、Swarm、Compose 相关功能调试

### 启动开发环境

```bash
./develop.sh
```

开发脚本会自动：

- **后端**：复制 `config.yml` 为 `.local.yml`（如不存在），并以**中控模式**启动（`ISRVD_MODE=center CONFIG_PATH=.local.yml go run ./server/cmd/server`），便于调试节点管理与节点切换；需要单机模式时用 `ISRVD_MODE=server ./develop.sh`
- **前端**：进入 `webview`，安装依赖并执行 `npm run dev`
- **端口清理**：启动前尝试释放 `8080` 和 `3000` 端口
- **代理**：前端开发服务器（`3000`）把 `/api/`、`/openapi/` 和节点视角的 `/n/` 代理到 `8080`；`/n/<节点ID>/` 下的页面由后端内嵌的前端产物提供，修改前端后需 `npm run build` 才能在节点视角下看到

Windows 环境可使用：

```bat
develop.bat
```

Windows 脚本直接使用根目录的 `config.yml` 以单机模式启动后端，并执行 `npm run dev`；需先在 `webview` 目录安装前端依赖。

### 构建与校验

GitHub Actions 从 `go.mod` 读取 Go 版本，CNB 使用 `golang:1.26-bookworm` 构建镜像。若设置了 `GOTOOLCHAIN=local`，本地已安装的 Go 必须满足 `go.mod` 的最低版本要求。

```bash
# 完整分发构建
./build.sh

# 后端全包编译检查
go test ./...

# Go 静态检查
go vet ./...

# 前端类型检查
(cd webview && npm run lint)

# 前端格式与 import 排序检查
(cd webview && npm run format:check)

# 前端样式一致性检查
(cd webview && python3 scripts/review-style.py)

# 空白与冲突标记检查（在仓库根目录执行）
git diff --check
```

> 贡献代码前请优先阅读 [AGENTS.md](AGENTS.md)。该文件是当前仓库的代码规范与协作约定入口，旧版 `CODE_STYLE` 不再作为规范来源。

## 架构设计

分层边界、包级依赖方向与设计原则见 [AGENTS.md](AGENTS.md)「3) 项目架构」；前端专项规范见 [webview/AGENTS.md](webview/AGENTS.md)。

## 安全特性

- JWT 认证，敏感字段（密钥、密码）不返回前端；SSH 密码/私钥加密落盘
- 文件路径校验，防止目录遍历攻击
- 解压校验路径，防止 Zip Slip 攻击
- WebSocket 连接需经过认证中间件
- 基于路由的细粒度权限控制，路由访问级别支持 `0` 需权限、`1` 需登录、`-1` 匿名
- 操作审计日志，路由审计级别支持 `0` 按 Method、`-1` 忽略、`1` 强制记录，默认记录非 GET 请求与 WebSocket 连接
- 支持可限制代理来源 CIDR 的信任 Header 认证（`tha.headerName`），`tha.trustedCIDRs` 默认为本机回环地址

## 许可证

本项目基于 **Apache License 2.0** 发布，详见 [LICENSE](LICENSE)。

第三方组件协议详见 [NOTICE](NOTICE)。
