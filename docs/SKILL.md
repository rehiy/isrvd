---
name: isrvd-ops
description: 通过 isrvd API 进行容器部署、服务管理、镜像操作、路由配置、文件管理等运维操作。当用户要求"部署服务"、"管理容器"、"拉取/推送镜像"、"配置路由"、"管理 Swarm"、"Compose 部署"、"文件管理"、"Web 终端"等运维任务时使用此 Skill。
---

# isrvd 运维 Skill

## 快速开始

推荐按 **Python → Node.js** 的顺序选择脚本；仅当 Python 和 Node.js 都不可用时，才使用 Bash 版兜底。三者共用 `~/.config/isrvd/profile.json`。

**首选：Python 标准库版**（无 `curl`/`jq` 依赖）

```bash
python3 ./scripts/api.py token "$ISRVD_APIURL" "$ISRVD_APITOKEN"
python3 ./scripts/api.py get "/docker/containers"
python3 ./scripts/api.py post "/docker/container" '{"image":"...","name":"..."}'
```

**备选：Node.js 内置库版**（无 `curl`/`jq` 依赖）

```bash
node ./scripts/api.js token "$ISRVD_APIURL" "$ISRVD_APITOKEN"
node ./scripts/api.js get "/docker/containers"
node ./scripts/api.js post "/docker/container" '{"image":"...","name":"..."}'
```

**兜底兼容：Bash 版**（仅当 `python3` 与 `node` 都不可用时使用；保留给旧文档中的 `isrvd_get` / `source` 交互式用法）

```bash
source ./scripts/api.sh

# 认证方式（优先使用环境变量，自动保存到 ~/.config/isrvd/profile.json）
isrvd_token "$ISRVD_APIURL" "$ISRVD_APITOKEN"
# 或
isrvd_login "$ISRVD_APIURL" "$ISRVD_USERNAME" "$ISRVD_PASSWORD"

isrvd_get "/docker/containers"
isrvd_post "/docker/container" '{"image":"...","name":"..."}'
```

脚本默认提取统一响应中的 `.payload`；数组对象会自动转为紧凑表格，降低输出噪音。Python/Node 版支持简单 selector（如 `.content`、`.username`）。Bash 版只作为 Python/Node 都不可用时的兜底兼容方案。

**⚠️ 操作规范（必须遵守）：**
1. **禁止硬编码**：不要假设任何 IP、端口、路径、容器名——全部通过 API 查询或环境变量获取
2. **禁止 base64**：不要用 base64 编码内容写入文件，使用 `isrvd_post "/filer/modify"` 或 `isrvd_upload`
3. **filer 路径 ≠ 宿主机路径**：volume mount 的 hostPath 必须是宿主机真实路径，先通过 inspect isrvd 容器确认映射关系，见 [references/system/filer.md](references/system/filer.md)
4. **不要重复重建容器**：静态文件更新直接写 filer 即可，容器无需重建；只有初次部署或更换镜像时才需要重建

---

## 文档索引

按使用场景查找对应文档：

| 模块 | 文档 | 说明 |
|------|------|------|
| 概览 | [references/overview.md](references/overview.md) | 服务探测、监控数据 |
| 容器 | [references/docker/containers.md](references/docker/containers.md) | 容器 CRUD、日志、终端 |
| 镜像 | [references/docker/images.md](references/docker/images.md) | 镜像搜索/拉取/构建/推送 |
| 网络 | [references/docker/networks.md](references/docker/networks.md) | Docker 网络管理 |
| 数据卷 | [references/docker/volumes.md](references/docker/volumes.md) | Docker 数据卷管理 |
| 仓库 | [references/docker/registries.md](references/docker/registries.md) | 镜像仓库配置 |
| Swarm | [references/swarm/info.md](references/swarm/info.md) | 集群、节点、令牌 |
| Swarm | [references/swarm/services.md](references/swarm/services.md) | 服务部署/扩缩容/更新 |
| Swarm | [references/swarm/tasks.md](references/swarm/tasks.md) | 任务列表 |
| Compose | [references/compose.md](references/compose.md) | Docker Compose / Swarm Stack 部署与重部署 |
| APISIX | [references/apisix/routes.md](references/apisix/routes.md) | 路由 CRUD、启用/禁用 |
| APISIX | [references/apisix/upstreams.md](references/apisix/upstreams.md) | 上游、负载均衡 |
| APISIX | [references/apisix/consumers.md](references/apisix/consumers.md) | Consumer、白名单 |
| APISIX | [references/apisix/ssl.md](references/apisix/ssl.md) | SSL 证书、PluginConfig |
| Caddy | [references/caddy/routes.md](references/caddy/routes.md) | 路由 CRUD（反向代理/静态文件） |
| Caddy | [references/caddy/certs.md](references/caddy/certs.md) | TLS 证书管理 |
| Caddy | [references/caddy/config.md](references/caddy/config.md) | 全局配置、原始 JSON 配置 |
| 系统 | [references/system/config.md](references/system/config.md) | 系统配置、审计日志 |
| 系统 | [references/system/account.md](references/system/account.md) | 登录、成员管理、API Token |
| 系统 | [references/system/filer.md](references/system/filer.md) | 文件管理、上传下载、压缩解压 |
| 系统 | [references/system/cron.md](references/system/cron.md) | 计划任务 |
| 系统 | [references/system/ssh.md](references/system/ssh.md) | SSH 主机/凭据管理、SFTP、SSH 终端 |
| 终端 | [references/shell.md](references/shell.md) | Web Shell（本地终端） |
| Agent | [references/agent.md](references/agent.md) | Agent 代理（OpenAI 兼容 API） |

> 📌 各文档中包含完整的 API 端点列表、请求/响应字段说明和示例。

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
│   ├── 路由/消费者       → references/apisix/routes.md 或 references/apisix/consumers.md
│   └── Caddy 路由        → references/caddy/routes.md (DELETE /caddy/route/:index)
│
└── 管理
    ├── 镜像仓库         → references/docker/registries.md
    ├── 成员/权限/Token/OIDC/TOTP 2FA → references/system/account.md
    ├── 文件管理         → references/system/filer.md
    ├── 计划任务         → references/system/cron.md
    ├── SSH 主机管理     → references/system/ssh.md
    ├── Shell 终端       → references/shell.md (GET /shell WebSocket)
    └── WebSSH 终端     → references/system/ssh.md (GET /ssh/to/:id WebSocket)
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
# APISIX
isrvd_get "/apisix/routes"
isrvd_post "/apisix/route" '{"name":"<NAME>","uri":"<URI>","status":1,"upstream":{"type":"roundrobin","nodes":{"<HOST>:<PORT>":1}}}'

# APISIX（为已有路由配置 Consumer 白名单；缺失 Consumer 需先创建并配置 key-auth）
ROUTE_ID=<ROUTE_ID>  # 从 isrvd_get "/apisix/routes" 结果中获取
isrvd_post "/apisix/consumer" '{"username":"<NEW_USERNAME>","plugins":{"key-auth":{"key":"<AUTH_KEY>"}}}'
isrvd_post "/apisix/whitelist" '{"route_id":"'"$ROUTE_ID"'","consumers":["<USERNAME>","<NEW_USERNAME>"],"key_auth":{"header":"apikey","query":"apikey","hide_credentials":false}}'

# Caddy（反向代理）
isrvd_post "/caddy/route" '{
  "match": {"hosts": ["<DOMAIN>"], "paths": ["/*"]},
  "handler": {"kind": "reverse_proxy", "upstreams": ["<HOST>:<PORT>"]}
}'
```

### 临时禁用/启用 APISIX 路由

```bash
isrvd_patch "/apisix/route/<ROUTE_ID>/status" '{"status":0}'
isrvd_patch "/apisix/route/<ROUTE_ID>/status" '{"status":1}'
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
