# Swarm 集群信息与节点 API

## 集群信息

```bash
isrvd_get "/swarm/info"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| clusterID | string | 集群 ID |
| createdAt | string | 创建时间（RFC3339） |
| nodes | number | 节点总数 |
| managers | number | Manager 数 |
| workers | number | Worker 数 |
| services | number | 服务总数 |
| tasks | number | 任务总数 |

## 列出节点

```bash
isrvd_get "/swarm/nodes"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 节点 ID |
| hostname | string | 主机名 |
| role | string | `manager` / `worker` |
| availability | string | `active` / `pause` / `drain` |
| state | string | `ready` / `down` |
| addr | string | IP 地址 |
| engineVersion | string | Docker 引擎版本 |
| leader | boolean | 是否为 Leader |

## 查看节点详情

```bash
isrvd_get "/swarm/node/NODE_ID"
```

额外字段：`os, architecture, cpus, memoryBytes, labels, createdAt, updatedAt`

## 查看节点运行中的服务

```bash
isrvd_get "/swarm/node/NODE_ID/services"
```

需要权限 `GET /api/swarm/node/:id/services`。节点详情页在具有此权限时展示服务列表，支持桌面表格和移动端卡片；刷新同时更新节点与服务信息。

返回按服务名排序的数组，仅包含在该节点有实际 `running` 任务的服务；历史、失败、等待调度及已完成任务不计入。正在停止但实际仍为 `running` 的任务仍计入。无运行服务时返回 `[]`。节点不存在或 Docker 查询失败返回 `500`，Swarm 不可用返回 `503`。

| 字段 | 类型 | 说明（均只读） |
|------|------|------|
| id | string | 服务 ID |
| name | string | 服务名 |
| image | string | 服务配置的镜像（滚动更新期间可能与旧任务镜像不同） |
| mode | string | `replicated` / `global` |
| replicas | number \| null | 服务在整个集群的期望副本数；global 为 `null` |
| runningTasks | number | **当前节点**实际运行的任务数，不是集群总数 |
| ports | object[] \| null | 服务端口配置 `{protocol, targetPort, publishedPort, publishMode}`；无端口为 `null` |
| createdAt | string | 服务创建时间（RFC3339） |
| updatedAt | string | 服务更新时间（RFC3339） |

结果反映 Swarm manager 的任务状态，用于查看服务分布，不代表实时 CPU/内存利用率。任务与服务分两次读取，并非原子快照；查询期间被删除的服务不会返回。

## 获取加入令牌

```bash
isrvd_get "/swarm/token"
```

返回：`{"worker":"SWMTKN-...","manager":"SWMTKN-..."}`

## 节点操作

```bash
isrvd_post "/swarm/node/NODE_ID/action" '{"action":"active"}'
isrvd_post "/swarm/node/NODE_ID/action" '{"action":"pause"}'
isrvd_post "/swarm/node/NODE_ID/action" '{"action":"drain"}'
isrvd_post "/swarm/node/NODE_ID/action" '{"action":"remove"}'
```

`remove` 为强制移除（节点仍在线也会被移除）。
