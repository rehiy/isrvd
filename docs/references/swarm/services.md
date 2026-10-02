# Swarm 服务 API

创建服务、扩缩容和强制更新总是启用 registry 查询以解析镜像 tag/digest；仅当服务镜像的 registry host 匹配 `docker.registries[].url` 且该仓库配置了用户名和密码时，才会以 Swarm 原生 registry auth 方式附带认证，便于 Worker 节点拉取私有仓库镜像。Docker Hub 镜像（含 `library/nginx` 这类无 host 形式）不匹配认证。

> Swarm 不可用时 `/swarm/*` 返回 `503`；请求体绑定失败返回 `400`，其余错误返回 `500`。

## 列出服务

```bash
isrvd_get "/swarm/services"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 服务 ID |
| name | string | 服务名 |
| image | string | 镜像 |
| mode | string | `replicated` / `global` |
| replicas | number \| null | 副本数；global 模式为 `null` |
| runningTasks | number | 运行中的任务数 |
| ports | object[] \| null | `{protocol, targetPort, publishedPort, publishMode}`；无端口时为 `null` |
| createdAt | string | 创建时间 |
| updatedAt | string | 更新时间 |

## 查看服务详情

```bash
isrvd_get "/swarm/service/<SVC_ID>"
```

返回完整 `ServiceDetail`，在列表字段基础上额外包含：`env、args、networks、mounts、labels、constraints`。

## 创建服务

```bash
isrvd_post "/swarm/service" '{
  "name": "<NAME>",
  "image": "<IMAGE>",
  "replicas": <N>,
  "ports": [{"targetPort": <PORT>, "publishedPort": <PORT>, "protocol": "tcp"}],
  "mounts": [{"type": "bind", "source": "<HOST_PATH>", "target": "<CONTAINER_PATH>"}],
  "constraints": ["node.hostname == <NODE_HOSTNAME>"]
}'
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✅ | 服务名 |
| image | string | ✅ | 镜像 |
| mode | string | | `replicated`（默认）/ `global`；非 `global` 的值一律按 `replicated` |
| replicas | number | | 副本数；不传或传 `0` 均按 `1`（创建时不能建 0 副本服务） |
| env | string[] | | 环境变量 |
| args | string[] | | 命令参数 |
| networks | string[] | | 网络名 |
| ports | object[] | | `{targetPort, publishedPort, protocol, publishMode}` |
| mounts | object[] | | `{type, source, target, readOnly}` |
| labels | object | | 标签 |
| constraints | string[] | | 放置约束；指定运行节点时使用 `node.hostname == <NODE_HOSTNAME>` |

说明：创建时会按 `image` 自动匹配已配置的 Docker 仓库认证，并请求 registry 解析镜像 tag/digest。返回 `{"id": "<SVC_ID>"}`。

## 服务操作

`action` 仅支持 `scale` / `remove` / `force-update`，其他值返回"不支持的操作"。

## 扩缩容

```bash
isrvd_post "/swarm/service/<SVC_ID>/action" '{"action":"scale","replicas":<N>}'
```

说明：`replicas` 必填（非负整数，可为 `0`），缺失时返回"不支持的操作: scale"；仅 replicated 模式服务支持扩缩容。扩缩容会沿用服务当前镜像自动匹配仓库认证，并请求 registry 解析镜像 tag/digest。

## 删除服务

```bash
isrvd_post "/swarm/service/<SVC_ID>/action" '{"action":"remove"}'
```

## 强制更新（重新部署）

```bash
isrvd_post "/swarm/service/<SVC_ID>/action" '{"action":"force-update"}'
```

说明：强制更新会沿用服务当前镜像自动匹配仓库认证，并请求 registry 解析镜像 tag/digest。

## 服务日志

```bash
isrvd_get "/swarm/service/<SVC_ID>/logs?tail=100"
```

返回：`{"logs": ["timestamped log line", ...]}`。快照最多 4MB，超出截断并追加 `[output truncated at 4194304 bytes]`。

实时日志使用 SSE：

```bash
GET /api/swarm/service/<SVC_ID>/logs/stream?tail=100&token=<JWT>
```

响应类型为 `text/event-stream`。服务端会先输出最近 `tail` 行，然后持续推送服务所有任务的聚合日志；每 25 秒发送 `event: heartbeat`，出错时发送 `event: error`；断开 SSE 连接即停止跟随。
