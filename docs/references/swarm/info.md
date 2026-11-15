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
| state | string | `ready` / `down` / `disconnected` / `unknown` |
| addr | string | IP 地址 |
| engineVersion | string | Docker 引擎版本 |
| leader | boolean | 是否为 Leader |

## 查看节点详情

```bash
isrvd_get "/swarm/node/NODE_ID"
```

除节点列表字段外，还包含以下字段（均只读）：

| 字段 | 类型 | 说明 |
|------|------|------|
| version | number | Docker 节点版本，保存标签时原样提交，用于并发校验 |
| os | string | 操作系统 |
| architecture | string | CPU 架构 |
| cpus | number | CPU 核心数 |
| memoryBytes | number | 内存字节数 |
| labels | object \| null | 节点标签键值表，值为字符串，允许空字符串 |
| createdAt | string | 创建时间（RFC3339） |
| updatedAt | string | 更新时间（RFC3339） |

## 管理节点标签

在连接 Swarm manager 的 Web 界面，进入「Swarm → 节点 → 节点详情 → 节点标签 → 编辑标签」，添加、修改或删除键值行后保存。没有标签时也显示编辑入口；只有读取权限的用户可以查看标签，编辑需要 `PUT /api/swarm/node/:id/labels` 权限。写操作按默认审计规则记录。

标签用于表达节点属性，可被多个服务共用。例如两台广州节点都设置 `region=guangzhou`，服务使用 `node.labels.region == guangzhou` 约束。标签修改或删除可能触发服务重新调度。

先获取节点详情中的 `version`，再提交完整标签集合：

```bash
isrvd_get "/swarm/node/NODE_ID"
# 将 version 换为刚读取的节点版本；保留仍需要的其他标签
isrvd_put "/swarm/node/NODE_ID/labels" '{"version":123,"labels":{"region":"guangzhou","cfs":"true"}}'
```

| 请求字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| version | number | 是 | 读取详情时的节点版本 |
| labels | object | 是 | 完整标签集合；未提交的标签将删除；`{}` 清空全部标签，不能为 `null` |

标签键不能为空或包含首尾空白，值必须为字符串，可以为空。删除部分标签时，在提交集合中移除对应的键。清空标签示例：

```bash
# 使用最新读取的 version
isrvd_put "/swarm/node/NODE_ID/labels" '{"version":124,"labels":{}}'
```

成功返回 `200`，不携带 payload。非法请求返回 `400`，节点不存在返回 `404`，版本冲突返回 `409`（刷新详情后重新编辑），Docker 调用失败返回 `500`，Swarm 模块不可用返回 `503`。更新仅替换节点标签，保留节点角色、可用性及其他配置。

## 查看节点运行中的服务

```bash
isrvd_get "/swarm/node/NODE_ID/services"
```

需要权限 `GET /api/swarm/node/:id/services`。

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
