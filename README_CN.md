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
| 集中管理 | 同一个二进制通过 `ISRVD_MODE` 作为单机、中控或受管机运行，一个控制台管理多台服务器，详见 [集中管理](#集中管理) |
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

安装拆为两个独立脚本，无需互相下载或加载；均可使用 `--auto`（默认）、`--cn`、`--global` 选择软件源：

- **系统服务 + 组件容器（`isrvd-system.sh`）**：isrvd 以 systemd 服务运行；默认不选网关，直接执行 `install`、不传 `--components` 即仅安装 isrvd。可按需用 `--components caddy|apisix` 选择网关：`caddy` 安装独立 Caddy 容器、不带 etcd，`apisix` 自动安装独立 APISIX 和 etcd 容器。首页再次点击已选网关即可取消选择；`--components none` 仅保留为旧命令兼容。新安装不支持独立 etcd 或逗号分隔的组件组合；历史部署中所属组件仍可更新和卸载。组件使用 host 网络，管理接口和 etcd 仅监听 `127.0.0.1`，数据默认保存在 `/srv/data`，安装时可用 `--data PATH` 自定义组件数据目录。不选择网关时无需 Docker，也不接受 `--data`。
- **全容器化（`isrvd-docker.sh`）**：自动准备 Docker、单节点 Swarm 和可挂载的 overlay 网络 `sdnet`，运行所选 `isrvd:tag` 一体化镜像。默认 tag 为 `slim`；内置 `caddy`、`apisix` 标签自动匹配端口布局，自定义标签用 `--variant slim|caddy|apisix` 指定布局。

```bash
# 仅安装系统服务（默认），无需 Docker
bash <(curl -fsSL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) install

# 系统服务 + Caddy（自定义组件数据目录）；或 APISIX + etcd（默认 /srv/data）
bash <(curl -fsSL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) install --components caddy --data /mnt/isrvd-data
bash <(curl -fsSL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) install --components apisix

# 全容器化：slim / caddy / apisix 按需选择
bash <(curl -fsSL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-docker.sh) install --tag slim
bash <(curl -fsSL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-docker.sh) install --tag caddy
bash <(curl -fsSL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-docker.sh) install --tag apisix

# 自定义 tag：将 YOUR_PUBLISHED_TAG 替换为所选仓库中已经发布的标签
bash <(curl -fsSL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-docker.sh) install --tag YOUR_PUBLISHED_TAG --variant apisix
```

`isrvd-system.sh install --caddy` / `--apisix` 现在选择独立组件；需要一体化容器请使用 `isrvd-docker.sh install --caddy` / `--apisix`。`--systemd`、`--docker` 在各自脚本中可省略。Docker 版容器名为 `isrvd`，独立组件名为 `isrvd-caddy`、`isrvd-apisix`、`isrvd-etcd`。

组件数据目录须为非根绝对路径，不能包含 `.`、`..` 路径段、冒号、反斜杠或控制字符，也不能与 `/usr/local/isrvd` 安装目录重叠；脚本用 `realpath` 解析路径，组件子目录及配置文件不能通过符号链接重定向。Caddy 使用该目录下的 `caddy`（证书等存储）和 `caddy-config`（运行配置），etcd 使用 `etcd`，组件启动配置位于 `conf/components`；isrvd 主配置仍为 `/usr/local/isrvd/config.yml`。`--data` 仅用于新安装，不会迁移已有数据。

更新和卸载必须使用对应脚本，不再自动切换部署方式：`isrvd-system.sh update` 更新二进制及所属组件，从所属容器的实际 bind 挂载恢复组件数据目录，挂载不一致时在停服前中止；`isrvd-docker.sh update` 重建主容器。保留已有配置和角色；Docker 更新保留镜像 tag，也可用 `isrvd-docker.sh update --tag YOUR_PUBLISHED_TAG --variant apisix` 更换标签；容器替换失败时恢复旧容器。组件沿用安装时固定的镜像版本，不自动跨版本升级。`isrvd-system.sh uninstall` 删除系统服务及所属组件，保留实际组件数据目录；`isrvd-docker.sh uninstall` 只删除主容器，保留 `/srv/data`。系统服务卸载前请先备份 `/usr/local/isrvd/config.yml`。

在线入口 `/install/`（或 `?script=systemd`）提供系统服务脚本，`/install/?script=docker` 提供全容器化脚本，二者独立缓存。发布包包含两份脚本；旧的本地安装器可能不识别新包内的脚本文件，请使用最新在线脚本执行更新。

系统服务 + APISIX 可通过环境变量 `ISRVD_APISIX_ADMIN_KEY` 提供 32-128 个字母、数字、`_` 或 `-` 组成的非纯数字密钥；未设置时随机生成。脚本把同一密钥通过环境传入 APISIX 容器，并写入仅 root 可读的 `/usr/local/isrvd/config.yml`（0600）中 `apisix.adminKey`，管理地址为 `http://127.0.0.1:9180/apisix/admin`；不会打印密钥或依赖 Go 环境变量覆盖。更新沿用已有组件密钥，修复此前遗漏的管理路径和密钥配置，不读取安装器环境来轮换密钥。请保护配置文件及其备份。组件启动配置位于所选数据目录的 `conf/components`（默认 `/srv/data/conf/components`）；APISIX 就绪检查会验证带密钥的 Admin API，保留配置与传入密钥不匹配时安装会失败并回滚。

系统服务安装在解压前完成密钥、组件镜像和归档预检；后续失败或可捕获中断会回滚本次创建的二进制、链接、服务及所属容器，保留组件数据，允许重新安装。组件更新若因强制终止留下 `isrvd-组件-update-backup`，后续更新会在停服前中止；请确认备份归属并手动恢复，或运行 `isrvd-system.sh uninstall` 清理所属正常及备份容器后重装。内置 `slim` / `latest` / `caddy` / `apisix` 标签必须匹配对应的端口布局；安装及更新都会拒绝不一致的 `--variant`，自定义标签可显式指定布局。

脚本从安装器环境读取 `ISRVD_MODE`（默认 `server`，可选 `center`、`agent`），请在执行 `install` 前导出；受管机还需导出 `ISRVD_CENTER_URL` 和 `ISRVD_NODE_TOKEN`。安装时这些值会写入仅 root 可读的 systemd drop-in（`installer.conf`）或传入 Docker 容器。中控在配置文件中启用端口转发（`portmap.range`，如 `20000-20999`，以及 `portmap.bind`）；Docker 中控需设置 `portmap.bind: 0.0.0.0` 并执行 `update`，脚本会发布 `portmap.range` 范围的 TCP 与 UDP 端口（范围宜小）。Docker 受管机不发布 8080 端口。更新时保留已安装的角色；如需更换角色，请重建 Docker 容器，或修改 systemd 环境后重启。

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
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) install
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) install --cn

# 更新/卸载/仅下载
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) update
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) update --global
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) uninstall
bash <(curl -sL https://jscdn.rehi.org/gh/rehiy/isrvd/build/script/isrvd-system.sh) download
```

二进制更新会先下载并校验安装包，再停止服务，并保留 `config.yml`。Docker 更新会保留旧容器，直到新容器通过启动检查；失败时恢复旧容器。若更新中断后遗留 `isrvd-update-backup` 容器，需先处理该容器再重试。执行 `bash isrvd-system.sh --help` 查看可用命令与选项。

配置文件位置通过 `ISRVD_CONFIG` 指定，详见 [配置](#配置)。

## 集中管理

同一个二进制，通过环境变量 `ISRVD_MODE` 选择角色，一个控制台管理多台服务器：

| 模式 | 作用 |
| ------ | ------ |
| `server`（默认） | 单机 |
| `center` | 中控：提供 Web 界面、账号与权限，接收受管机接入，并把请求按节点转发 |
| `agent` | 受管机：不开放任何对外端口，主动连接中控，中控转发来的请求在本机执行 |

```text
浏览器 ──> center ──┬── 页面、账号、权限、审计 ──> 本进程内的 isrvd
                    └── /n/<节点ID>/api/… ──隧道──> agent（本进程内的 isrvd）
                                                  ▲ 受管机主动出站，中控从不主动外连
```

- **无需入站端口**：受管机只需要能访问中控，位于 NAT 或防火墙后也可以接入；断线后按指数退避自动重连
- **切换节点即切换视角**：有权限的成员在页面头部选择节点后，Docker、Swarm、Compose、文件、本机进程、终端、计划任务、APISIX、Caddy 和系统概览都作用于该节点，包括实时日志、终端和大文件上传下载
- **账号与配置在中控**：成员、权限、系统配置、AI 助手和 SSH 远程管理始终由中控处理，节点上的操作使用中控的登录态
- **按需出现**：节点切换器和侧边栏「集中管控」在 `center` 模式且当前用户有 `GET /api/node/nodes` 权限时显示

### 接入节点

1. 在中控进程环境中设置 `ISRVD_MODE=center`，重启服务。
2. 在中控的节点管理页点击「接入节点」，填写必填名称并创建节点。保存 token，并复制弹窗中的安装命令；节点此时为「待接入」状态。
3. 在受管机进程环境中设置：

```bash
export ISRVD_MODE=agent
export ISRVD_CENTER_URL=https://center.example.com
export ISRVD_NODE_TOKEN="<中控生成的 token>"
```

4. 重启后直接用 token 连接。token 无时效限制，接入后同一 token 可用于重启与重连。已在线时拒绝重复连接（409），不会顶替旧会话。

中控支持 `tha`，网关按 `tha.trustedCIDRs` 校验真实连接来源；受管模式不能启用 `tha`。部署参数由进程环境提供，YAML、etcd 与配置 API 管理业务配置。保存 token 的环境文件应限制权限为 `0600`。

### 容器部署

提前准备完整的 `/srv/data/conf/isrvd.yml` 保存业务配置，通过 `-e` 传入部署参数，中控与受管机分别部署在各自机器上：

```bash
# 中控（8080 为对外入口）
docker run -d --name isrvd-center -e ISRVD_MODE=center --network sdnet -p 8080:8080 \
  -v /srv/data:/data -v /var/run/docker.sock:/var/run/docker.sock rehiy/isrvd:slim

# 受管机（不需要 -p）
docker run -d --name isrvd-agent --network sdnet \
  -e ISRVD_MODE=agent -e ISRVD_CENTER_URL=https://center.example.com \
  -e ISRVD_NODE_TOKEN="<中控生成的 token>" \
  -v /srv/data:/data -v /var/run/docker.sock:/var/run/docker.sock rehiy/isrvd:slim
```

受管机不能启用 `tha`；受管模式下 `listenAddr` 不生效。受管机至少需要一个创始人成员，否则转发请求返回 401。切换后只能经中控访问，地址或 token 错误时，需修改受管机环境变量并重启。

### systemd 部署

通过 `systemctl edit isrvd` 在 `[Service]` 中设置 `Environment="ISRVD_MODE=center"`；受管机还需设置 `ISRVD_CENTER_URL` 和 `ISRVD_NODE_TOKEN`。保存后执行 `systemctl daemon-reload` 与 `systemctl restart isrvd`。安装脚本默认以单机模式启动，如需其他角色，请在执行 `install` 前导出相应的部署环境变量。

### 安全与边界

- 节点管理按 `/api/node/...` 路由权限授权；业务转发先在中控校验成员的业务路由权限，再以受管机本地服务令牌执行。创始人拥有全部权限。授权对所有节点生效，当前按路由而非节点 ID 分配权限
- 接入 token 无时效限制。创建节点时生成 86 位字母数字随机 token，仅显示一次；`node-list.yml` 只保存其 SHA-256 哈希和状态（`pending`、`active`、`revoked`），首次认证通过后节点变为已激活。token 遗失时需重新创建节点。节点可随时吊销，吊销后立即断开
- 节点管理操作、节点上的写操作与终端会话都会记入中控的审计页面（操作人为中控登录用户），受管机本机的审计页面里则记录为其本地创始人
- 中控为单实例设计，重启时节点短暂断开后自动重连；版本无需一致，兼容性由协议版本与 features 协商决定（见 [docs/multi-node.md](docs/multi-node.md)）
- 完整的参数表、接口与行为说明见 [docs/multi-node.md](docs/multi-node.md)，接口字段见 [受管节点](docs/references/node/nodes.md)

## 配置

### 配置来源

配置加载器统一去除环境变量值的首尾空白，空白值使用默认值。

配置位置由 `ISRVD_CONFIG` 指定，支持本地 YAML 与 etcd（value 仍为 config.yml 同款 YAML）：

```bash
# 默认读取 ./config.yml
./isrvd

# 本地 YAML
ISRVD_CONFIG=/data/conf/isrvd.yml ./isrvd

# etcd
etcdctl put /isrvd/config "$(cat /data/conf/isrvd.yml)"
ISRVD_CONFIG="etcd://user:pass@127.0.0.1:2379/isrvd/config?scheme=http&timeout=5s" ./isrvd

# etcd key 不存在时，用 fallback YAML 初始化并写入 etcd
ISRVD_CONFIG="etcd://127.0.0.1:2379/isrvd/config?fallback=/data/conf/isrvd.yml" ./isrvd

# etcd 完整配置示例
# etcd://user:pass@host1:2379,host2:2379/key?scheme=http&timeout=5s&fallback=/path/config.yml
```

**etcd** 认证可省略，也可用 `ETCD_USERNAME` / `ETCD_PASSWORD` 补充或覆盖 URI 中的认证信息。etcd key 发生 PUT 变更时，isrvd 会重载配置、注册中心和业务服务。计划任务、SSH 主机与凭据等业务数据同样存入 etcd（key 为 `<配置 key>/cron.yml` 等），业务数据 key 缺失时导入 `rootDirectory` 下的同名文件，后续读写使用 etcd，本地来源文件作为导入副本保留；审计与监控日志写本地。通过系统配置 API 保存的本地 YAML 也会立即触发重载；若直接在磁盘上修改 YAML，则需发送 `SIGHUP` 或重启进程。

### 配置项

各配置段的含义见 [配置段说明](docs/references/system/config.md#配置段说明)。

## 权限

权限基于路由进行细粒度控制。默认的 `AccessPerm` 路由要求成员持有对应的完整路由权限，Founder 不受此限制；`AccessAuth` 路由只要登录即可访问，`AccessAnon` 路由允许匿名访问。

**权限格式**：`<METHOD> /api/<模块>/<路由>`（如 `GET /api/docker/containers`、`POST /api/compose/docker`）

**前端权限判断**：使用 `portal.hasPerm('<METHOD> /api/<路由>')` 控制按钮/操作的显示

> 留空 = 无 `AccessPerm` 路由权限；具体可用路由可在登录后通过 `GET /api/account/routes` 获取。
>
> 集中管理的节点接口由中控网关提供。中控模式下，节点管理权限列入 `/account/routes`，可在成员管理中分配；`/n/<节点ID>/api/...` 转发请求检查对应的业务路由权限，授权对所有节点生效。

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

- **后端**：复制 `config.yml` 为 `.local.yml`（如不存在），以 `ISRVD_MODE=center ISRVD_CONFIG=.local.yml go run ./server/cmd/server` 启动，开发时即可调试节点管理与切换
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

> 贡献代码前请优先阅读 [AGENTS.md](AGENTS.md)。该文件定义仓库的代码规范与协作约定。

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
