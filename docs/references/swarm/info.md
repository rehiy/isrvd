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
