# Swarm 任务 API

## 列出任务

`serviceID` 可选，不传时返回集群内全部任务。返回 `Task[]`：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 任务 ID |
| serviceID | string | 服务 ID |
| serviceName | string | 服务名；对应服务已不存在时为空串 |
| nodeID | string | 节点 ID |
| nodeName | string | 节点主机名；对应节点已不存在时为空串 |
| slot | number | 任务槽位 |
| image | string | 镜像 |
| state | string | Docker 任务状态，如 `running` / `pending` / `failed` / `complete` / `shutdown` / `rejected` 等 |
| message | string | 状态信息 |
| err | string | 错误信息 |
| updatedAt | string | 更新时间 |

```bash
isrvd_get "/swarm/tasks?serviceID=<SVC_ID>"
```
