---
name: isrvd-ops
description: 通过 isrvd API 进行容器部署、服务管理、镜像操作、路由配置、文件管理等运维操作。当用户要求"部署服务"、"管理容器"、"拉取/推送镜像"、"配置路由"、"管理 Swarm"、"Compose 部署"、"文件管理"、"Web 终端"等运维任务时使用此 Skill。
---

# isrvd 运维 Skill

```bash
source ./scripts/api.sh
# 认证方式（优先使用环境变量，自动保存到 ~/.config/isrvd/profile.json）
isrvd_token "$ISRVD_APIURL" "$ISRVD_APITOKEN"
# 或 isrvd_login "$ISRVD_APIURL" "$ISRVD_USERNAME" "$ISRVD_PASSWORD"
```

调用：`isrvd_get "/path"` / `isrvd_post "/path" '{body}'`，输出紧凑 JSON（数组自动转表格）。按需用 jq 自行处理返回值。

**⚠️ 操作规范（必须遵守）：**
1. **禁止硬编码**：不要假设任何 IP、端口、路径、容器名——全部通过 API 查询或环境变量获取
2. **禁止 base64**：不要用 base64 编码内容写入文件，使用 `isrvd_post "/filer/modify"` 或 `isrvd_upload`
3. **filer 路径 ≠ 宿主机路径**：volume mount 的 hostPath 必须是宿主机真实路径，先通过 inspect isrvd 容器确认映射关系，见 docs/system/filer.md
4. **不要重复重建容器**：静态文件更新直接写 filer 即可，容器无需重建；只有初次部署或更换镜像时才需要重建

---

## API 文档索引（按需读取，勿全部加载）

### Docker

| 文档 | 覆盖内容 |
|------|----------|
| [docs/docker/containers.md](docs/docker/containers.md) | 容器列表、创建、操作、日志、stats、终端 |
| [docs/docker/images.md](docs/docker/images.md) | 镜像列表、详情、搜索、构建、标签、拉取、推送、删除、清理（prune） |
| [docs/docker/networks.md](docs/docker/networks.md) | 网络列表、详情、创建、删除 |
| [docs/docker/volumes.md](docs/docker/volumes.md) | 数据卷列表、详情、创建、删除 |
| [docs/docker/registries.md](docs/docker/registries.md) | 镜像仓库 CRUD |

### Swarm

| 文档 | 覆盖内容 |
|------|----------|
| [docs/swarm/info.md](docs/swarm/info.md) | 集群信息、节点列表/详情/操作、加入令牌 |
| [docs/swarm/services.md](docs/swarm/services.md) | 服务列表、创建、扩缩容、强制更新、日志 |
| [docs/swarm/tasks.md](docs/swarm/tasks.md) | 任务列表 |

### Compose

| 文档 | 覆盖内容 |
|------|----------|
| [docs/compose.md](docs/compose.md) | Docker Compose 与 Swarm Stack 的部署、读取配置、重部署（含外部 Compose 标签聚合接管、全量重建与按服务更新镜像）、forcePull 强制拉取 |

### APISIX

| 文档 | 覆盖内容 |
|------|----------|
| [docs/apisix/routes.md](docs/apisix/routes.md) | 路由 CRUD、启用/禁用 |
| [docs/apisix/upstreams.md](docs/apisix/upstreams.md) | 上游 CRUD、负载均衡配置 |
| [docs/apisix/consumers.md](docs/apisix/consumers.md) | Consumer CRUD、白名单管理 |
| [docs/apisix/ssl.md](docs/apisix/ssl.md) | SSL 证书、PluginConfig、插件列表 |

### Caddy

| 文档 | 覆盖内容 |
|------|----------|
| [docs/caddy/routes.md](docs/caddy/routes.md) | 路由 CRUD（match + handler，含反向代理/文件服务/静态响应/原始 handle） |
| [docs/caddy/certs.md](docs/caddy/certs.md) | TLS 证书 CRUD（磁盘文件 / 内联 PEM / 自动签发三种来源） |
| [docs/caddy/config.md](docs/caddy/config.md) | 概览、全局选项（Admin/日志/端口/优雅关闭）、获取/整体替换原始 JSON 配置 |

### 系统

| 文档 | 覆盖内容 |
|------|----------|
| [docs/system/overview.md](docs/system/overview.md) | 服务探测、系统资源统计、监控历史数据 |
| [docs/system/config.md](docs/system/config.md) | 系统配置、审计日志 |
| [docs/system/account.md](docs/system/account.md) | 登录、OIDC 登录、成员管理、权限、API Token |
| [docs/system/filer.md](docs/system/filer.md) | 文件 CRUD、上传下载（含 inline 预览）、压缩解压 |
| [docs/system/cron.md](docs/system/cron.md) | 计划任务 CRUD、立即执行、启用/禁用、执行历史 |
| [docs/system/webssh.md](docs/system/webssh.md) | SSH 主机管理（密码/私钥认证）、WebSocket 终端会话 |
| Agent 代理 | `ANY /api/agent/*path` 代理到配置的 OpenAI 兼容 LLM API，自动注入 `agent.apiKey` 并可重写 `agent.model` |

---

## 决策树

```
用户需求
├── 部署/创建
│   ├── 单个容器        → docs/docker/containers.md
│   ├── 多容器应用(单机) → docs/compose.md §1
│   ├── 集群服务(Stack)  → docs/compose.md §2
│   ├── 集群服务(单服务) → docs/swarm/services.md
│   ├── 配置 APISIX 路由 → docs/apisix/routes.md
│   └── 配置 Caddy 路由  → docs/caddy/routes.md
│
├── 更新/变更
│   ├── 更新/接管 Compose 项目 → docs/compose.md (project 标签聚合、redeploy + serviceName/image)
│   ├── 更新容器镜像     → docs/docker/images.md (拉取) + docs/docker/containers.md (重建)
│   ├── 扩缩容           → docs/swarm/services.md
│   ├── 重新部署         → docs/swarm/services.md (force-update)
│   ├── 修改路由/上游    → docs/apisix/routes.md 或 docs/apisix/upstreams.md
│   ├── 修改 Caddy 路由  → docs/caddy/routes.md（更新触发 /load 整体替换）
│   ├── 修改 Caddy 全局选项 → docs/caddy/config.md (PUT /caddy/global)
│   └── 修改系统配置     → docs/system/config.md
│
├── 查询/监控
│   ├── 容器/镜像/网络/卷 → docs/docker/ 下对应文件
│   ├── 集群/服务/任务    → docs/swarm/ 下对应文件
│   ├── 路由/上游/插件    → docs/apisix/ 下对应文件
│   ├── Caddy 路由/配置   → docs/caddy/ 下对应文件
│   ├── 系统状态          → docs/system/overview.md
│   ├── 监控历史数据      → docs/system/overview.md
│   ├── 日志             → docs/docker/containers.md 或 docs/swarm/services.md
│   └── 文件管理         → docs/system/filer.md
│
├── 删除/清理
│   ├── 容器/镜像/网络/卷 → docs/docker/ 下对应文件（action=remove）
│   ├── Swarm 服务        → docs/swarm/services.md（action=remove）
│   ├── 路由/消费者       → docs/apisix/routes.md 或 docs/apisix/consumers.md
│   └── Caddy 路由        → docs/caddy/routes.md (DELETE /caddy/route/:index)
│
└── 管理
    ├── 镜像仓库         → docs/docker/registries.md
    ├── 成员/权限/Token/OIDC 登录 → docs/system/account.md
    ├── 文件管理         → docs/system/filer.md
    ├── 计划任务         → docs/system/cron.md
    ├── SSH 主机管理     → docs/system/webssh.md
    └── Web 终端         → GET /api/shell (WebSocket) 或 GET /api/ssh/terminal/:id (WebSocket)
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
isrvd_get "/overview/status"
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

> filer 路径是 isrvd 内部路径，非宿主机路径。初次创建容器挂载卷时，需先查出宿主机真实路径（见 docs/system/filer.md），之后文件更新不再需要重建容器。

### 其他文件操作

```bash
isrvd_get "/filer/list?path=<DIR>"
isrvd_get "/filer/read?path=<FILE>" '.content'
isrvd_post "/filer/modify" '{"path":"<FILE>","content":"<CONTENT>"}'
isrvd_upload "/filer/upload" "file" "<LOCAL_FILE>" "path=<FILER_DIR>"
```
