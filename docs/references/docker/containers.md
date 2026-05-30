# Docker 容器 API

## Docker 服务信息

```bash
isrvd_get "/docker/info"
```

返回 Docker daemon 基础信息。

## 列出容器

```bash
isrvd_get "/docker/containers"
isrvd_get "/docker/containers?all=true"
```

返回 `ContainerInfo[]`：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 容器短 ID（12位） |
| name | string | 容器名 |
| image | string | 镜像名 |
| state | string | `running` / `exited` / `paused` / `created` / `restarting` / `dead` |
| status | string | 状态描述（如 "Up 2 hours"） |
| ports | string[] | 端口映射（如 `"0.0.0.0:8080->80/tcp"`） |
| networks | string[] | 所属网络名 |
| created | number | 创建时间戳（Unix 秒） |
| isSwarm | boolean | 是否为 Swarm 管理的容器 |
| isSelf | boolean | 是否为当前 isrvd 所在容器；用于前端隐藏危险操作 |
| labels | object | 标签键值对；`com.docker.compose.project` / `com.docker.compose.service` 可用于判断是否属于 Compose 项目 |

## 查看容器详情

```bash
isrvd_get "/docker/container/<CONTAINER_ID>"
```

返回 `ContainerDetail`，包含基础信息、运行配置、端口映射、挂载、环境变量、标签等字段。

## 创建容器

```bash
isrvd_post "/docker/container" '{
  "image": "<IMAGE>",
  "name": "<NAME>",
  "ports": {"<HOST_PORT>": "<CONTAINER_PORT>"},
  "env": ["<KEY>=<VALUE>"],
  "volumes": [{"hostPath": "<HOST_PATH>", "containerPath": "<CONTAINER_PATH>", "readOnly": true}],
  "network": "<NETWORK>",
  "restart": "unless-stopped",
  "memory": 256,
  "cpus": 0.5
}'
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| image | string | ✅ | 镜像名 |
| name | string | ✅ | 容器名 |
| cmd | string[] | | 启动命令 |
| env | string[] | | 环境变量（`KEY=VALUE`） |
| ports | object | | `{"宿主端口": "容器端口"}` |
| volumes | object[] | | `{hostPath, containerPath, readOnly}`，**hostPath 必须是宿主机真实路径**，filer 显示的路径是 isrvd 内部路径，不可直接使用 |
| network | string | | 网络名（先通过 API 查询已有网络） |
| restart | string | | `no` / `always` / `on-failure` / `unless-stopped` |
| memory | number | | 内存限制（MB） |
| cpus | number | | CPU 限制（核数） |
| workdir | string | | 工作目录 |
| user | string | | 运行用户 |
| hostname | string | | 主机名 |
| privileged | boolean | | 特权模式 |
| capAdd | string[] | | 添加的 Capabilities |
| capDrop | string[] | | 移除的 Capabilities |
| labels | object | | 容器标签；Compose 部署会自动写入 `com.docker.compose.project` / `com.docker.compose.service` |

## 容器操作

当目标容器被识别为当前 isrvd 所在容器时，后端会拒绝 `stop` / `restart` / `remove` / `pause`，避免误操作导致服务中断。

```bash
isrvd_post "/docker/container/<CONTAINER_ID>/action" '{"action":"start"}'
isrvd_post "/docker/container/<CONTAINER_ID>/action" '{"action":"stop"}'
isrvd_post "/docker/container/<CONTAINER_ID>/action" '{"action":"restart"}'
isrvd_post "/docker/container/<CONTAINER_ID>/action" '{"action":"remove"}'
isrvd_post "/docker/container/<CONTAINER_ID>/action" '{"action":"pause"}'
isrvd_post "/docker/container/<CONTAINER_ID>/action" '{"action":"unpause"}'
```

## 容器日志

```bash
isrvd_get "/docker/container/<CONTAINER_ID>/logs?tail=100"
```

返回：`{id, logs: string[]}`

实时日志使用 SSE：

```bash
GET /api/docker/container/<CONTAINER_ID>/logs/stream?tail=100&token=<JWT>
```

响应类型为 `text/event-stream`。服务端会先输出最近 `tail` 行，然后持续推送新日志；断开 SSE 连接即停止跟随。

## 容器资源统计

```bash
isrvd_get "/docker/container/<CONTAINER_ID>/stats"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| cpuPercent | number | CPU 使用率 (%) |
| memoryUsage | number | 内存使用（字节） |
| memoryLimit | number | 内存限制（字节） |
| memoryPercent | number | 内存使用率 (%) |
| networkRx | number | 网络接收（字节） |
| networkTx | number | 网络发送（字节） |
| blockRead | number | 磁盘读取（字节） |
| blockWrite | number | 磁盘写入（字节） |
| pids | number | 进程数 |

## 容器终端

WebSocket 连接，不通过 harness 调用：

```bash
GET /api/docker/container/:id/exec?shell=/bin/sh
```

用于打开容器的交互式终端会话。
