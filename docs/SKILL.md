---
name: isrvd-ops
description: 通过 isrvd API 进行容器部署、服务管理、镜像操作、路由配置、文件管理等运维操作。当用户要求"部署服务"、"管理容器"、"拉取/推送镜像"、"配置路由"、"管理 Swarm"、"Compose 部署"、"文件管理"、"Web 终端"等运维任务时使用此 Skill。
---

# isrvd 运维 Skill

## 快速开始

```bash
source ./scripts/api.sh

# 认证方式（优先使用环境变量，自动保存到 ~/.config/isrvd/profile.json）
isrvd_token "$ISRVD_APIURL" "$ISRVD_APITOKEN"
# 或
isrvd_login "$ISRVD_APIURL" "$ISRVD_USERNAME" "$ISRVD_PASSWORD"
```

调用：`isrvd_get "/path"` / `isrvd_post "/path" '{body}'`，输出紧凑 JSON（数组自动转表格）。按需用 jq 自行处理返回值。

**⚠️ 操作规范（必须遵守）：**
1. **禁止硬编码**：不要假设任何 IP、端口、路径、容器名——全部通过 API 查询或环境变量获取
2. **禁止 base64**：不要用 base64 编码内容写入文件，使用 `isrvd_post "/filer/modify"` 或 `isrvd_upload`
3. **filer 路径 ≠ 宿主机路径**：volume mount 的 hostPath 必须是宿主机真实路径，先通过 inspect isrvd 容器确认映射关系，见 [references/system/filer.md](references/system/filer.md)
4. **不要重复重建容器**：静态文件更新直接写 filer 即可，容器无需重建；只有初次部署或更换镜像时才需要重建

---

## API 文档索引

### Overview（概览与监控）

| 文档 | 覆盖内容 |
|------|----------|
| [references/overview.md](references/overview.md) | 服务探测、系统资源统计、监控历史数据（支持实时/1h/6h/12h/24h 时间区间） |

**API 端点：**
- `GET /overview/probe` — 探测服务可用性
- `GET /overview/monitor?type=host|container&since=3600&id=<容器ID>` — 获取监控数据（since=0 为实时模式）
- `POST /overview/upgrade` — 升级程序至最新版本

---

### Docker（容器管理）

| 文档 | 覆盖内容 |
|------|----------|
| [references/docker/containers.md](references/docker/containers.md) | 容器列表、详情、创建、操作（start/stop/restart/remove/exec）、日志、stats、终端 |
| [references/docker/images.md](references/docker/images.md) | 镜像列表、详情、搜索、构建、标签、拉取、推送、删除、清理（prune） |
| [references/docker/networks.md](references/docker/networks.md) | 网络列表、详情、创建、删除 |
| [references/docker/volumes.md](references/docker/volumes.md) | 数据卷列表、详情、创建、删除 |
| [references/docker/registries.md](references/docker/registries.md) | 镜像仓库 CRUD |

**API 端点：**
- **信息**：`GET /docker/info`
- **容器**：`GET /docker/containers`、`GET /docker/container/:id`、`POST /docker/container`、`GET /docker/container/:id/stats`、`POST /docker/container/:id/action`、`GET /docker/container/:id/logs`、`GET /docker/container/:id/logs/stream`（WebSocket）、`GET /docker/container/:id/exec`
- **镜像**：`GET /docker/images`、`GET /docker/images/search`、`POST /docker/image/:id/action`、`POST /docker/image/:id/tag`、`GET /docker/image/:id`、`POST /docker/image/build`、`POST /docker/image/prune`、`POST /docker/image/push`、`POST /docker/image/pull`
- **网络**：`GET /docker/networks`、`POST /docker/network/:id/action`、`POST /docker/network`、`GET /docker/network/:id`
- **数据卷**：`GET /docker/volumes`、`POST /docker/volume/:name/action`、`POST /docker/volume`、`GET /docker/volume/:name`
- **镜像仓库**：`GET /docker/registries`、`POST /docker/registry`、`PUT /docker/registry`、`DELETE /docker/registry`

---

### Swarm（集群管理）

| 文档 | 覆盖内容 |
|------|----------|
| [references/swarm/info.md](references/swarm/info.md) | 集群信息、节点列表/详情/操作、加入令牌 |
| [references/swarm/services.md](references/swarm/services.md) | 服务列表、详情、创建、扩缩容、强制更新、日志 |
| [references/swarm/tasks.md](references/swarm/tasks.md) | 任务列表 |

**API 端点：**
- **信息**：`GET /swarm/info`
- **节点**：`GET /swarm/nodes`、`GET /swarm/node/:id`、`POST /swarm/node/:id/action`、`GET /swarm/token`
- **服务**：`GET /swarm/services`、`GET /swarm/service/:id`、`POST /swarm/service`、`POST /swarm/service/:id/action`、`POST /swarm/service/:id/force-update`、`GET /swarm/service/:id/logs`
- **任务**：`GET /swarm/tasks`

---

### Compose（多容器部署）

| 文档 | 覆盖内容 |
|------|----------|
| [references/compose.md](references/compose.md) | Docker Compose 与 Swarm Stack 的部署、读取配置、重部署（含外部 Compose 标签聚合接管、全量重建与按服务更新镜像）、forcePull 强制拉取 |

**API 端点：**
- `GET /compose/docker/:name` — 读取 Docker Compose 配置
- `GET /compose/swarm/:name` — 读取 Swarm Stack 配置
- `POST /compose/docker/deploy` — 部署 Docker Compose 应用
- `POST /compose/swarm/deploy` — 部署 Swarm Stack 应用
- `POST /compose/docker/:name/redeploy` — 重新部署 Docker Compose 应用（支持按服务更新镜像）
- `POST /compose/swarm/:name/redeploy` — 重新部署 Swarm Stack 应用（支持按服务更新镜像）

---

### APISIX（API 网关）

| 文档 | 覆盖内容 |
|------|----------|
| [references/apisix/routes.md](references/apisix/routes.md) | 路由 CRUD、启用/禁用（status patch） |
| [references/apisix/upstreams.md](references/apisix/upstreams.md) | 上游 CRUD、负载均衡配置 |
| [references/apisix/consumers.md](references/apisix/consumers.md) | Consumer CRUD、白名单管理 |
| [references/apisix/ssl.md](references/apisix/ssl.md) | SSL 证书 CRUD、PluginConfig 管理、插件列表 |

**API 端点：**
- **路由**：`GET /apisix/routes`、`GET /apisix/route/:id`、`POST /apisix/route`、`PUT /apisix/route/:id`、`PATCH /apisix/route/:id/status`、`DELETE /apisix/route/:id`
- **消费者**：`GET /apisix/consumers`、`POST /apisix/consumer`、`PUT /apisix/consumer/:username`、`DELETE /apisix/consumer/:username`
- **白名单**：`GET /apisix/whitelist`、`POST /apisix/whitelist/revoke`
- **插件配置**：`GET /apisix/plugin-configs`、`GET /apisix/plugin-config/:id`、`POST /apisix/plugin-config`、`PUT /apisix/plugin-config/:id`、`DELETE /apisix/plugin-config/:id`
- **上游**：`GET /apisix/upstreams`、`GET /apisix/upstream/:id`、`POST /apisix/upstream`、`PUT /apisix/upstream/:id`、`DELETE /apisix/upstream/:id`
- **SSL 证书**：`GET /apisix/ssls`、`GET /apisix/ssl/:id`、`POST /apisix/ssl`、`PUT /apisix/ssl/:id`、`DELETE /apisix/ssl/:id`
- **插件列表**：`GET /apisix/plugins`

---

### Caddy（Web 服务器）

| 文档 | 覆盖内容 |
|------|----------|
| [references/caddy/routes.md](references/caddy/routes.md) | 路由 CRUD（match + handler，含反向代理/文件服务/静态响应/原始 handle） |
| [references/caddy/certs.md](references/caddy/certs.md) | TLS 证书 CRUD（磁盘文件 / 内联 PEM / 自动签发三种来源） |
| [references/caddy/config.md](references/caddy/config.md) | 概览、全局选项（Admin/日志/端口/优雅关闭）、获取/整体替换原始 JSON 配置 |

**API 端点：**
- **概览与配置**：`GET /caddy/info`、`GET /caddy/config`、`POST /caddy/config`
- **全局选项**：`GET /caddy/global`、`PUT /caddy/global`
- **路由**：`GET /caddy/routes`、`GET /caddy/route/:index`、`POST /caddy/route`、`PUT /caddy/route/:index`、`DELETE /caddy/route/:index`
- **TLS 证书**：`GET /caddy/certs`、`POST /caddy/cert`、`PUT /caddy/cert/:key`、`DELETE /caddy/cert/:key`

---

### System（系统管理）

| 文档 | 覆盖内容 |
|------|----------|
| [references/overview.md](references/overview.md) | 服务探测、系统资源统计、监控历史数据 |
| [references/system/config.md](references/system/config.md) | 系统配置（GET/PUT）、审计日志 |
| [references/system/account.md](references/system/account.md) | 登录（账号密码/OIDC）、成员管理、权限、API Token |
| [references/system/filer.md](references/system/filer.md) | 文件 CRUD、上传下载（含 inline 预览）、压缩解压、目录大小计算 |
| [references/system/cron.md](references/system/cron.md) | 计划任务 CRUD、立即执行、启用/禁用、执行历史、可用脚本类型 |
| [references/ssh/hosts.md](references/ssh/hosts.md) | SSH 主机管理（密码/私钥认证） |
| [references/ssh/sftp.md](references/ssh/sftp.md) | SFTP 文件管理（列出/上传/下载/删除/重命名/权限） |
| [references/shell.md](references/shell.md) | Web Shell 终端（WebSocket） |
| [references/agent.md](references/agent.md) | Agent 代理（OpenAI 兼容 LLM API） |
| [references/shell.md](references/shell.md) | Web Shell 终端（WebSocket） |
| | [references/agent.md](references/agent.md) | Agent 代理（OpenAI 兼容 LLM API） |

**API 端点：**
- **配置**：`GET /system/config`、`PUT /system/config`
- **审计日志**：`GET /system/audit/logs`
- **认证信息**：`GET /account/info`
- **登录**：`POST /account/login`、`GET /account/oidc/login`、`GET /account/oidc/callback`、`POST /account/oidc/exchange`
- **Passkey 登录**（无需认证）：`POST /account/passkey/login/begin`、`POST /account/passkey/login/finish`
- **Passkey 管理**（需认证）：`POST /account/passkey/register/begin`、`POST /account/passkey/register/finish`、`GET /account/passkey/credentials`、`PUT /account/passkey/credential/:credentialID`（重命名）、`DELETE /account/passkey/credential/:credentialID`
- **凭证管理**：`POST /account/token`、`PUT /account/password`
- **路由权限**：`GET /account/routes`
- **成员管理**：`GET /account/members`、`POST /account/member`、`PUT /account/member/:username`、`DELETE /account/member/:username`
- **文件管理**：`GET /filer/list`、`GET /filer/read`、`GET /filer/download`、`GET /filer/dir-size`、`POST /filer/mkdir`、`POST /filer/create`、`POST /filer/modify`、`POST /filer/rename`、`POST /filer/delete`、`POST /filer/chmod`、`POST /filer/upload`、`POST /filer/zip`、`POST /filer/unzip`
- **计划任务**：`GET /cron/types`、`GET /cron/jobs`、`POST /cron/jobs`、`PUT /cron/jobs/:id`、`DELETE /cron/jobs/:id`、`POST /cron/jobs/:id/run`、`POST /cron/jobs/:id/enable`、`GET /cron/jobs/:id/logs`
- **Web Shell**：`GET /shell`（WebSocket）
- **SSH 主机**：`GET /ssh/hosts`、`POST /ssh/host`、`PUT /ssh/host/:id`、`DELETE /ssh/host/:id`
- **SSH 终端**：`GET /ssh/to/:id`（WebSocket）
- **SFTP**：`GET /ssh/sftp/:id/ls`、`GET /ssh/sftp/:id/read`、`GET /ssh/sftp/:id/download`、`POST /ssh/sftp/:id/upload`、`POST /ssh/sftp/:id/rm`、`POST /ssh/sftp/:id/mkdir`、`POST /ssh/sftp/:id/rename`、`POST /ssh/sftp/:id/chmod`、`POST /ssh/sftp/:id/chown`、`GET /ssh/sftp/:id/dir-size`
- **Agent 代理**：`ANY /agent/*path`

---

## 决策树

```
用户需求
├── 部署/创建
│   ├── 单个容器        → references/docker/containers.md
│   ├── 多容器应用(单机) → references/compose.md §1
│   ├── 集群服务(Stack)  → references/compose.md §2
│   ├── 集群服务(单服务) → references/swarm/services.md
│   ├── 配置 APISIX 路由 → references/apisix/routes.md
│   └── 配置 Caddy 路由  → references/caddy/routes.md
│
├── 更新/变更
│   ├── 更新/接管 Compose 项目 → references/compose.md (project 标签聚合、redeploy + serviceName/image)
│   ├── 更新容器镜像     → references/docker/images.md (拉取) + references/docker/containers.md (重建)
│   ├── 扩缩容           → references/swarm/services.md
│   ├── 重新部署         → references/swarm/services.md (force-update)
│   ├── 修改路由/上游    → references/apisix/routes.md 或 references/apisix/upstreams.md
│   ├── 修改 Caddy 路由  → references/caddy/routes.md（更新触发 /load 整体替换）
│   ├── 修改 Caddy 全局选项 → references/caddy/config.md (PUT /caddy/global)
│   └── 修改系统配置     → references/system/config.md
│
├── 查询/监控
│   ├── 容器/镜像/网络/卷 → references/docker/ 下对应文件
│   ├── 集群/服务/任务    → references/swarm/ 下对应文件
│   ├── 路由/上游/插件    → references/apisix/ 下对应文件
│   ├── Caddy 路由/配置   → references/caddy/ 下对应文件
│   ├── 系统状态          → references/overview.md
│   ├── 监控历史数据      → references/overview.md (since=3600|21600|43200|86400)
│   ├── 日志             → references/docker/containers.md 或 references/swarm/services.md
│   └── 文件管理         → references/system/filer.md
│
├── 删除/清理
│   ├── 容器/镜像/网络/卷 → references/docker/ 下对应文件（action=remove）
│   ├── Swarm 服务        → references/swarm/services.md（action=remove）
│   ├── 路由/消费者       → references/apisix/routes.md 或 docs/apisix/consumers.md
│   └── Caddy 路由        → references/caddy/routes.md (DELETE /caddy/route/:index)
│
└── 管理
    ├── 镜像仓库         → references/docker/registries.md
    ├── 成员/权限/Token/OIDC 登录 → references/system/account.md
    ├── 文件管理         → references/system/filer.md
    ├── 计划任务         → references/system/cron.md
│   ├── SSH 主机管理     → references/ssh/hosts.md
│   ├── SFTP 文件管理    → references/ssh/sftp.md
│   ├── Shell 终端       → references/shell.md (GET /shell WebSocket)
│   └── WebSSH 终端     → references/ssh/hosts.md (GET /ssh/to/:id WebSocket)
```

---

## 配置重载

isrvd 支持运行时重载，无需重启进程：

- **etcd 配置变更**：自动触发重载，无需手动操作
- **SIGHUP 信号**：手动触发，适用于本地文件配置场景

```bash
kill -HUP $(pgrep isrvd)
```

触发行为：重新加载配置 → 重新初始化 registry 客户端 → 重新初始化各业务服务，服务恢复后 API 立即生效。

### 服务不可用时的行为

服务初始化失败时，对应模块路由返回 `503`：

```json
{"error": "apisix service unavailable", "module": "apisix", "reload": "send SIGHUP to reload services"}
```

---

## 常见工作流

> 以下示例中的路径、IP、端口、容器名等**仅为格式参考**，实际值必须通过 API 查询获取。

### 健康检查

```bash
isrvd_get "/overview/probe"
isrvd_get "/overview/monitor?type=host&since=3600"
isrvd_get "/docker/containers"
isrvd_get "/swarm/services"
```

### 拉取镜像并创建容器

```bash
isrvd_post "/docker/image/pull" '{"image":"<IMAGE>"}'
isrvd_post "/docker/container" '{"image":"<IMAGE>","name":"<NAME>","ports":{"<HOST_PORT>":"<CONTAINER_PORT>"},"restart":"unless-stopped"}'
isrvd_get "/docker/containers"
```

### 更新容器镜像

```bash
CID=<CONTAINER_ID>  # 从 isrvd_get "/docker/containers" 结果中获取
isrvd_post "/docker/image/pull" '{"image":"<NEW_IMAGE>"}'
isrvd_post "/docker/container/$CID/action" '{"action":"stop"}'
isrvd_post "/docker/container/$CID/action" '{"action":"remove"}'
isrvd_post "/docker/container" '{"image":"<NEW_IMAGE>","name":"<NAME>","ports":{"<HOST_PORT>":"<CONTAINER_PORT>"},"restart":"unless-stopped"}'
```

### Compose 部署

```bash
isrvd_post "/compose/docker/deploy" "$(jq -n --arg content "$(cat docker-compose.yml)" '{content:$content}')"
isrvd_post "/compose/swarm/deploy" "$(jq -n --arg content "$(cat stack.yml)" '{content:$content}')"
```

### 更新 Compose 服务镜像并重建

```bash
isrvd_post "/compose/docker/<NAME>/redeploy" '{"serviceName":"<SERVICE_NAME>","image":"<NEW_IMAGE>"}'
isrvd_post "/compose/swarm/<NAME>/redeploy" '{"serviceName":"<SERVICE_NAME>","image":"<NEW_IMAGE>"}'
```

### 部署 Swarm 服务并验证

```bash
isrvd_post "/swarm/service" '{"name":"<NAME>","image":"<IMAGE>","replicas":<N>,"ports":[{"targetPort":<PORT>,"publishedPort":<PORT>}]}'
isrvd_get "/swarm/services"
```

### 扩缩容

```bash
isrvd_post "/swarm/service/<SVC_ID>/action" '{"action":"scale","replicas":<N>}'
```

### 滚动更新

```bash
isrvd_post "/swarm/service/<SVC_ID>/force-update"
```

### 为新服务配置路由

```bash
# 先查现有路由
isrvd_get "/apisix/routes"
# 创建（upstream 的 nodes 地址通过查询实际容器/服务获取）
isrvd_post "/apisix/route" '{"name":"<NAME>","uri":"<URI>","status":1,"upstream":{"type":"roundrobin","nodes":{"<HOST>:<PORT>":1}}}'
```

### 临时禁用/启用路由

```bash
isrvd_patch "/apisix/route/<ROUTE_ID>/status" '{"status":0}'
isrvd_patch "/apisix/route/<ROUTE_ID>/status" '{"status":1}'
```

### 为新服务配置 Caddy 路由

```bash
# 反向代理（最常见）
isrvd_post "/caddy/route" '{
  "match": {"hosts": ["<DOMAIN>"], "paths": ["/*"]},
  "handler": {"kind": "reverse_proxy", "upstreams": ["<HOST>:<PORT>"]}
}'

# 查看当前所有 Caddy 路由
isrvd_get "/caddy/routes"

# 备份完整配置（推荐在重大变更前执行）
isrvd_get "/caddy/config" > /tmp/caddy-backup.json
```

### 更新静态文件（无需重建容器）

直接写 filer，已挂载该目录的容器立即生效：

```bash
# 先查看 filer 可用目录
isrvd_get "/filer/list?path=/"

# 写入小文件
isrvd_post "/filer/modify" '{"path":"<FILER_PATH>/<FILE>","content":"..."}'

# 写入大文件：先写到本地再上传（禁止用 base64）
cat > /tmp/<FILE> << 'EOF'
...内容...
EOF
isrvd_upload "/filer/upload" "file" "/tmp/<FILE>" "path=<FILER_PATH>"
```

> filer 路径是 isrvd 内部路径，非宿主机路径。初次创建容器挂载卷时，需先查出宿主机真实路径（见 references/system/filer.md），之后文件更新不再需要重建容器。

### 其他文件操作

```bash
isrvd_get "/filer/list?path=<DIR>"
isrvd_get "/filer/read?path=<FILE>" '.content'
isrvd_post "/filer/modify" '{"path":"<FILE>","content":"<CONTENT>"}'
isrvd_upload "/filer/upload" "file" "<LOCAL_FILE>" "path=<FILER_DIR>"
```
