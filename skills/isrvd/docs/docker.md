# Docker 管理 API

> 所有接口前缀: `/api/docker`  
> 只读操作需要 `docker:r` 权限，写操作需要 `docker:rw` 权限

---

## §1 Docker 信息

```
GET /api/docker/info
```

返回 Docker 引擎信息（版本、运行时、存储驱动等）。

---

## §2 容器管理

### §2.1 列出容器

```
GET /api/docker/containers?all=true|false
```

`all=true` 包含已停止的容器。返回 `ContainerInfo[]`：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 容器短 ID（12位） |
| name | string | 容器名 |
| image | string | 镜像名 |
| state | string | 状态: `running` / `exited` / `paused` / `created` / `restarting` / `dead` |
| status | string | 状态描述（如 "Up 2 hours"） |
| ports | string[] | 端口映射（如 `"0.0.0.0:8080->80/tcp"`） |
| networks | string[] | 所属网络名 |
| created | number | 创建时间戳（Unix 秒） |
| isSwarm | boolean | 是否为 Swarm 管理的容器 |
| labels | object | 标签键值对 |

### §2.2 创建容器

```
POST /api/docker/container/create
```

请求体 `ContainerCreateRequest`：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| image | string | ✅ | 镜像名（如 `nginx:latest`） |
| name | string | ✅ | 容器名 |
| cmd | string[] | | 启动命令 |
| env | string[] | | 环境变量（`KEY=VALUE` 格式） |
| ports | object | | 端口映射 `{"宿主端口": "容器端口"}` |
| volumes | VolumeMapping[] | | 目录映射 |
| network | string | | 网络名 |
| restart | string | | 重启策略: `no` / `always` / `on-failure` / `unless-stopped` |
| memory | number | | 内存限制（MB） |
| cpus | number | | CPU 限制（核数） |
| workdir | string | | 工作目录 |
| user | string | | 运行用户 |
| hostname | string | | 主机名 |
| privileged | boolean | | 特权模式 |
| capAdd | string[] | | 添加的 Linux Capabilities |
| capDrop | string[] | | 移除的 Linux Capabilities |

`VolumeMapping` 结构：

```json
{ "hostPath": "/host/path", "containerPath": "/container/path", "readOnly": false }
```

**完整示例**：

```json
{
  "image": "nginx:latest",
  "name": "my-nginx",
  "ports": { "8080": "80", "8443": "443" },
  "env": ["NGINX_HOST=example.com", "NGINX_PORT=80"],
  "volumes": [
    { "hostPath": "/data/nginx/html", "containerPath": "/usr/share/nginx/html", "readOnly": true },
    { "hostPath": "/data/nginx/conf", "containerPath": "/etc/nginx/conf.d", "readOnly": false }
  ],
  "network": "my-network",
  "restart": "unless-stopped",
  "memory": 256,
  "cpus": 0.5
}
```

### §2.3 容器操作

```
POST /api/docker/container/:id/action
```

| action | 说明 |
|--------|------|
| `start` | 启动容器 |
| `stop` | 停止容器 |
| `restart` | 重启容器 |
| `remove` | 删除容器 |
| `pause` | 暂停容器 |
| `unpause` | 恢复容器 |

示例：`{"action": "restart"}`

### §2.4 容器日志

```
GET /api/docker/container/:id/logs?tail=100&follow=false
```

| 参数 | 说明 |
|------|------|
| tail | 返回最后 N 行（默认 100） |
| follow | 是否持续跟踪（SSE 流） |

### §2.5 容器资源统计

```
GET /api/docker/container/:id/stats
```

返回 CPU、内存、网络 I/O 实时统计。

### §2.6 容器终端（WebSocket）

```
WS /ws/docker/exec?id=<容器ID>&shell=/bin/sh&token=<jwt-token>
```

---

## §3 镜像管理

### §3.1 列出镜像

```
GET /api/docker/images?all=false
```

### §3.2 查看镜像详情

```
GET /api/docker/image/:id
```

返回层信息、环境变量、CMD、Entrypoint、Labels 等。

### §3.3 搜索镜像

```
GET /api/docker/image/search/:term
```

从 Docker Hub 搜索镜像。

### §3.4 构建镜像

```
POST /api/docker/image/build
Body: { "dockerfile": "FROM nginx:latest\nCOPY . /app", "tag": "myimage:v1" }
```

### §3.5 镜像打标签

```
POST /api/docker/image/tag
Body: { "id": "镜像ID或名称", "repoTag": "newrepo:newtag" }
```

### §3.6 删除镜像

```
POST /api/docker/image/:id/action
Body: { "action": "remove" }
```

### §3.7 拉取镜像

```
POST /api/docker/registry/pull
Body: {
  "image": "nginx:latest",
  "registryUrl": "https://registry.example.com",  // 可选，空则 Docker Hub
  "namespace": "myns"                               // 可选
}
```

### §3.8 推送镜像

```
POST /api/docker/registry/push
Body: {
  "image": "本地镜像名:tag",
  "registryUrl": "https://registry.example.com",
  "namespace": "myns"  // 可选
}
```

---

## §4 镜像仓库管理

### §4.1 列出仓库

```
GET /api/docker/registries
```

### §4.2 添加仓库

```
POST /api/docker/registries
Body: {
  "name": "仓库名",
  "url": "https://registry.example.com",
  "username": "user",
  "password": "pass",
  "description": "描述"
}
```

### §4.3 更新仓库

```
PUT /api/docker/registries?url=原始URL
Body: { "name": "新名称", "url": "新URL", "username": "user", "password": "pass", "description": "描述" }
```

> ⚠️ 密码为空时保留原密码，不会清空。

### §4.4 删除仓库

```
DELETE /api/docker/registries?url=仓库URL
```

---

## §5 网络管理

### §5.1 列出网络

```
GET /api/docker/networks
```

### §5.2 查看网络详情

```
GET /api/docker/network/:id
```

### §5.3 创建网络

```
POST /api/docker/network/create
Body: { "name": "网络名", "driver": "bridge", "subnet": "172.20.0.0/16" }
```

### §5.4 删除网络

```
POST /api/docker/network/:id/action
Body: { "action": "remove" }
```

---

## §6 数据卷管理

### §6.1 列出数据卷

```
GET /api/docker/volumes
```

### §6.2 查看数据卷详情

```
GET /api/docker/volume/:name
```

### §6.3 创建数据卷

```
POST /api/docker/volume/create
Body: { "name": "卷名", "driver": "local" }
```

### §6.4 删除数据卷

```
POST /api/docker/volume/:name/action
Body: { "action": "remove" }
```

---

## 常见工作流

### 拉取镜像并创建容器

```bash
# 1. 拉取镜像
isrvd_post "/docker/registry/pull" '{"image":"nginx:latest"}'

# 2. 创建容器
isrvd_post "/docker/container/create" '{
  "image": "nginx:latest",
  "name": "web-server",
  "ports": {"80": "80"},
  "restart": "unless-stopped"
}'

# 3. 验证
isrvd_get "/docker/containers?all=false"
```

### 更新容器镜像

```bash
# 1. 拉取新版本
isrvd_post "/docker/registry/pull" '{"image":"nginx:1.25"}'

# 2. 停止并删除旧容器
isrvd_post "/docker/container/OLD_ID/action" '{"action":"stop"}'
isrvd_post "/docker/container/OLD_ID/action" '{"action":"remove"}'

# 3. 用新镜像创建容器（保持相同配置）
isrvd_post "/docker/container/create" '{"image":"nginx:1.25","name":"web-server",...}'
```
