# Swarm 任务 API

> 前缀: `/api/swarm`

## 列出任务

```
GET /api/swarm/tasks?serviceID=<SVC_ID>
```

返回 `Task[]`：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 任务 ID |
| serviceID | string | 服务 ID |
| serviceName | string | 服务名 |
| nodeID | string | 节点 ID |
| nodeName | string | 节点主机名 |
| slot | number | 任务槽位 |
| image | string | 镜像 |
| state | string | `running` / `pending` / `failed` / `complete` 等 |
| message | string | 状态信息 |
| err | string | 错误信息 |
| updatedAt | string | 更新时间 |

```bash
isrvd_get "/swarm/tasks?serviceID=<SVC_ID>"
```
