# Swarm 服务 API

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
| replicas | number | 副本数 |
| runningTasks | number | 运行中的任务数 |
| ports | object[] | `{protocol, targetPort, publishedPort, publishMode}` |
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
  "mounts": [{"type": "bind", "source": "<HOST_PATH>", "target": "<CONTAINER_PATH>"}]
}'
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✅ | 服务名 |
| image | string | ✅ | 镜像 |
| mode | string | | `replicated`（默认）/ `global` |
| replicas | number | | 副本数（默认 1） |
| env | string[] | | 环境变量 |
| args | string[] | | 命令参数 |
| networks | string[] | | 网络名 |
| ports | object[] | | `{targetPort, publishedPort, protocol, publishMode}` |
| mounts | object[] | | `{type, source, target, readOnly}` |
| labels | object | | 标签 |
| constraints | string[] | | 放置约束 |

## 扩缩容

```bash
isrvd_post "/swarm/service/<SVC_ID>/action" '{"action":"scale","replicas":<N>}'
```

## 删除服务

```bash
isrvd_post "/swarm/service/<SVC_ID>/action" '{"action":"remove"}'
```

## 强制更新（重新部署）

```bash
isrvd_post "/swarm/service/<SVC_ID>/force-update"
```

## 服务日志

```bash
isrvd_get "/swarm/service/<SVC_ID>/logs?tail=100"
```

返回：`{"logs": ["timestamped log line", ...]}`
