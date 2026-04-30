# Swarm 集群管理 API

> 所有接口前缀: `/api/swarm`  
> 只读操作需要 `swarm:r` 权限，写操作需要 `swarm:rw` 权限  
> Swarm 功能依赖 Docker 引擎可用

---

## §1 集群信息

```
GET /api/swarm/info
```

返回 Swarm 集群状态、管理节点数、工作节点数等信息。

---

## §2 节点管理

### §2.1 列出节点

```
GET /api/swarm/nodes
```

返回 `NodeInfo[]`：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 节点 ID |
| hostname | string | 主机名 |
| status | string | 状态: `ready` / `down` |
| availability | string | 可用性: `active` / `drain` / `pause` |
| role | string | 角色: `manager` / `worker` |
| engineVersion | string | Docker 引擎版本 |
| ip | string | 节点 IP |

### §2.2 查看节点详情

```
GET /api/swarm/node/:id
```

### §2.3 获取加入令牌

```
GET /api/swarm/join-tokens
```

返回 manager 和 worker 的加入令牌。

### §2.4 节点操作

```
POST /api/swarm/node/:id/action
```

| action | 说明 |
|--------|------|
| `active` | 设为活跃（接受新任务） |
| `drain` | 排空节点（迁移所有任务） |
| `pause` | 暂停调度（不接受新任务，现有任务保留） |

---

## §3 服务管理

### §3.1 列出服务

```
GET /api/swarm/services
```

返回 `ServiceInfo[]`：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 服务 ID |
| name | string | 服务名 |
| image | string | 镜像 |
| mode | string | `replicated` 或 `global` |
| replicas | number | 期望副本数 |
| runningTasks | number | 运行中的任务数 |
| ports | ServicePort[] | 端口映射 |
| createdAt | string | 创建时间 |
| updatedAt | string | 更新时间 |

### §3.2 查看服务详情

```
GET /api/swarm/service/:id
```

返回完整的服务规格（Spec）和当前状态。

### §3.3 创建服务

```
POST /api/swarm/service/create
```

请求体 `ServiceSpec`：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✅ | 服务名 |
| image | string | ✅ | 镜像名 |
| mode | string | | `replicated`（默认）或 `global` |
| replicas | number | | 副本数（replicated 模式，默认 1） |
| env | string[] | | 环境变量 `KEY=VALUE` |
| args | string[] | | 启动参数 |
| networks | string[] | | 网络名列表 |
| ports | ServicePort[] | | 端口映射 |
| mounts | ServiceMount[] | | 挂载配置 |
| labels | object | | 标签键值对 |
| constraints | string[] | | 调度约束（如 `node.role==manager`） |

`ServicePort`：

```json
{
  "protocol": "tcp",
  "targetPort": 80,
  "publishedPort": 8080,
  "publishMode": "ingress"
}
```

`ServiceMount`：

```json
{
  "type": "bind",
  "source": "/host/path",
  "target": "/container/path",
  "readOnly": false
}
```

> `type` 可选: `bind`（绑定挂载）、`volume`（命名卷）

**完整示例**：

```json
{
  "name": "web-service",
  "image": "nginx:latest",
  "mode": "replicated",
  "replicas": 3,
  "env": ["NGINX_HOST=example.com"],
  "ports": [
    { "protocol": "tcp", "targetPort": 80, "publishedPort": 8080, "publishMode": "ingress" }
  ],
  "mounts": [
    { "type": "bind", "source": "/data/html", "target": "/usr/share/nginx/html", "readOnly": true }
  ],
  "networks": ["my-overlay"],
  "constraints": ["node.role==worker"]
}
```

---

## §4 服务操作

### §4.1 扩缩容 / 删除

```
POST /api/swarm/service/:id/action
```

**扩缩容**（仅 `replicated` 模式）：

```json
{ "action": "scale", "replicas": 5 }
```

**删除服务**：

```json
{ "action": "remove" }
```

### §4.2 强制重新部署

```
POST /api/swarm/service/:id/redeploy
```

强制更新服务（拉取最新镜像并重新调度所有任务）。无需请求体。

### §4.3 查看服务日志

```
GET /api/swarm/service/:id/logs?tail=100
```

---

## §5 任务管理

### §5.1 列出任务

```
GET /api/swarm/tasks?serviceID=xxx
```

`serviceID` 可选，不传则返回所有任务。

返回任务列表，包含任务状态、所在节点、错误信息等。

---

## 常见工作流

### 创建并扩容服务

```bash
# 1. 创建服务
isrvd_post "/swarm/service/create" '{
  "name": "api-server",
  "image": "myapp:v1",
  "replicas": 2,
  "ports": [{"protocol":"tcp","targetPort":3000,"publishedPort":3000,"publishMode":"ingress"}],
  "env": ["NODE_ENV=production"]
}'

# 2. 查看服务状态
isrvd_get "/swarm/services"

# 3. 扩容到 5 个副本
isrvd_post "/swarm/service/SERVICE_ID/action" '{"action":"scale","replicas":5}'
```

### 滚动更新服务

```bash
# 强制重新部署（拉取最新镜像）
isrvd_post "/swarm/service/SERVICE_ID/redeploy"

# 查看任务状态确认更新进度
isrvd_get "/swarm/tasks?serviceID=SERVICE_ID"
```
