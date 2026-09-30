# Docker 容器 API

> Docker 不可用时，`/docker/*` 路由统一返回 `503`。请求体绑定失败返回 `400`，其余业务错误一般返回 `500`。

## Docker 服务信息

```bash
isrvd_get "/docker/info"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| containersRunning | number | 运行中容器数 |
| containersStopped | number | 已停止容器数（非 running/paused 均计入） |
| containersPaused | number | 已暂停容器数 |
| imagesTotal | number | 镜像总数 |
| volumesTotal | number | 数据卷总数 |
| networksTotal | number | 网络总数 |
| registryMirrors | string[] | daemon 配置的镜像加速地址 |
| indexServerAddress | string | 默认 Registry 地址 |

## 列出容器

```bash
isrvd_get "/docker/containers"
isrvd_get "/docker/containers?all=true"
# Docker 原生 filters 过滤（JSON 编码，需 URL 编码）
isrvd_get "/docker/containers?filters={\"status\":[\"exited\"]}"
isrvd_get "/docker/containers?filters={\"health\":[\"unhealthy\"]}"
isrvd_get "/docker/containers?filters={\"name\":[\"webshot\"]}"
isrvd_get "/docker/containers?filters={\"label\":[\"com.docker.compose.project\"]}"
```

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| all | boolean | 是否包含已停止的容器，默认 `false` |
| filters | string | Docker 原生 filters 的 JSON 编码，如 `{"health":["unhealthy"]}`；支持 `status` / `health` / `name` / `label` / `network` / `volume` / `ancestor` 等 Docker 标准过滤键；格式错误返回 400 |

返回 `ContainerInfo[]`：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 容器短 ID（12位） |
| name | string | 容器名 |
| image | string | 镜像名 |
| state | string | `running` / `exited` / `paused` / `created` / `restarting` / `dead` |
| status | string | 状态描述（如 "Up 2 hours"） |
| ports | string[] | 端口映射：`8080:80/tcp`；绑定指定 IP 时为 `127.0.0.1:8080:80/tcp`；未发布端口为 `80/tcp` |
| networks | string[] | 所属网络名；为空时不返回 |
| created | number | 创建时间戳（Unix 秒） |
| isSwarm | boolean | 是否为 Swarm 管理的容器；为 `false` 时不返回 |
| isSelf | boolean | 是否为当前 isrvd 所在容器；用于前端隐藏危险操作；为 `false` 时不返回 |
| labels | object | 标签键值对；`com.docker.compose.project` / `com.docker.compose.service` 可用于判断是否属于 Compose 项目；为空时不返回 |

## 查看容器详情

```bash
isrvd_get "/docker/container/<CONTAINER_ID>"
```

返回 `ContainerDetail`（包含创建请求的全部字段 + 运行态字段）：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 容器短 ID（12位） |
| name | string | 容器名 |
| state | string | 运行状态 |
| createdAt | string | 创建时间（Docker 返回的时间字符串） |
| image / cmd / env / workdir / user / hostname / privileged / capAdd / capDrop | - | 同创建请求 |
| ports | object | `{"[宿主IP:]宿主端口/协议": "容器端口"}` |
| network | string | NetworkMode |
| restart | string | 重启策略 |
| memory | number | 内存限制（MB） |
| cpus | number | CPU 限制（核数） |
| volumes | object[] | `{type, source, containerPath, readOnly}`，`source` 为宿主机路径（volume 类型时为宿主机上的挂载点） |
| labels | object | 容器标签 |
| autoRemove | boolean | 当前不回填，恒为 `false` |

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
| ports | object | | `{"[宿主IP:]宿主端口[/协议]": "容器端口"}`，宿主 IP 默认 `0.0.0.0`，协议默认 `tcp`（以键为准） |
| volumes | object[] | | `{type?, source?, hostPath?, containerPath, readOnly}`，见下方挂载说明 |
| network | string | | 网络名（先通过 API 查询已有网络） |
| restart | string | | `no`（默认）/ `always` / `on-failure` / `unless-stopped`；非法值按 `no` 处理 |
| memory | number | | 内存限制（MB） |
| cpus | number | | CPU 限制（核数） |
| workdir | string | | 工作目录 |
| user | string | | 运行用户 |
| hostname | string | | 主机名 |
| privileged | boolean | | 特权模式 |
| capAdd | string[] | | 添加的 Capabilities |
| capDrop | string[] | | 移除的 Capabilities |
| autoRemove | boolean | | 容器退出后自动删除 |
| labels | object | | 容器标签；Compose 部署会自动写入 `com.docker.compose.project` / `com.docker.compose.service` |

**挂载说明（`volumes[]`）：**

- `containerPath` 必填；`source` 优先于 `hostPath`，两者都为空时报错"挂载源不能为空"
- `type` 可选 `volume` / `bind`；不填时自动推断：绝对路径或以 `.` 开头为 `bind`，否则按卷名处理为 `volume`
- `volume` 类型的名称不能包含 `/`
- `bind` 类型需使用**宿主机真实路径**（filer 显示的是 isrvd 内部路径，不可直接使用）；相对路径在配置了 `docker.containerRoot` 时解析为 `<containerRoot>/<容器名>/<路径>`；挂载点不存在时自动创建

**行为：** 本地不存在镜像时先自动拉取；创建后立即启动，启动失败会强制删除刚创建的容器。返回 `{id, name}`（`id` 为 12 位短 ID）。

## 容器操作

`action` 可选 `start` / `stop` / `restart` / `remove` / `pause` / `unpause`，其他值返回"不支持的操作"。`stop` / `restart` 超时 10 秒；`remove` 为强制删除（运行中容器也会被删除）。

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

- `tail` 默认 `100`，可传 `all`；每行日志带时间戳
- 快照最多 4MB，超出截断并追加 `[output truncated at 4194304 bytes]`
- TTY 容器的日志合并为 `logs` 中的单个元素

实时日志使用 SSE：

```bash
GET /api/docker/container/<CONTAINER_ID>/logs/stream?tail=100&token=<JWT>
```

响应类型为 `text/event-stream`。服务端会先输出最近 `tail` 行，然后持续推送新日志；每 25 秒发送 `event: heartbeat`，获取容器信息或日志失败时发送 `event: error`；断开 SSE 连接即停止跟随。

## 容器资源统计

```bash
isrvd_get "/docker/container/<CONTAINER_ID>/stats"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 容器 ID |
| name | string | 容器名 |
| cpuPercent | number | CPU 使用率 (%) |
| cpuCores | number | 可用 CPU 核数 |
| cpuFreq | number | CPU 主频（MHz） |
| memoryUsage | number | 内存使用（字节，已扣除 `inactive_file` / `cache`） |
| memoryLimit | number | 内存限制（字节） |
| memoryPercent | number | 内存使用率 (%) |
| networkRx | number | 网络接收（字节） |
| networkTx | number | 网络发送（字节） |
| blockRead | number | 磁盘读取（字节） |
| blockWrite | number | 磁盘写入（字节） |
| pids | number | 进程数 |
| pidsLimit | number | 进程数上限 |
| cpuThrottled | object | `{periods, throttledPeriods, throttledTime}` |
| networkDetail | object | 按网卡名分组：`{rxBytes, rxPackets, rxErrors, rxDropped, txBytes, txPackets, txErrors, txDropped}` |
| blockDetail | object[] | `{major, minor, read, write}` |
| processList | object \| null | `{titles, processes}` |

## 容器终端

WebSocket 连接，不通过 harness 调用：

```bash
# 使用 wscat 连接（需先获取 token）
TOKEN=$(isrvd_login "$ISRVD_APIURL" "$ISRVD_USERNAME" "$ISRVD_PASSWORD" | jq -r '.payload.token')
wscat -c "ws://<HOST>/api/docker/container/<CONTAINER_ID>/exec?token=$TOKEN&shell=/bin/sh"
```

用于打开容器的交互式终端会话，`shell` 默认 `/bin/sh`。

前端窗口尺寸变化时应发送 resize 控制帧，格式为 `\u0000isrvd:resize:<cols>:<rows>`（`cols` 10–1000，`rows` 3–1000，超出范围的帧会被丢弃）；服务端会调用 Docker exec resize 同步容器 TTY 尺寸。

## 容器文件管理

> 通过 Docker SDK `CopyToContainer` / `CopyFromContainer` + exec 实现，支持列目录、上传下载、编辑、删除、重命名、创建目录、修改权限。

### 列出目录

```bash
isrvd_get "/docker/container/<ID>/file/ls?path=/"
```

`path` 默认 `/`。返回 `{path, files: ContainerFileInfo[]}`（容器内无 GNU `find` 时回退为 `ls -la`，此时 `modTime` 为 0）：

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 文件名 |
| size | number | 大小（字节） |
| mode | string | 权限字符串，如 `drwxr-xr-x` |
| modTime | number | 修改时间（Unix 时间戳） |
| isDir | boolean | 是否目录；软链接目标为目录时为 `true` |
| isLink | boolean | 是否软链接 |
| linkTarget | string | 软链接目标，仅软链接时返回 |

### 读取文件

```bash
isrvd_get "/docker/container/<ID>/file/read?path=/etc/hostname"
```

返回：`{content: string}`。`path` 必填（否则 400）；文件超过 4MB 返回 413；非 UTF-8 内容会被替换为 `\uFFFD`。

### 写入文件

```bash
isrvd_post "/docker/container/<ID>/file/write" '{"path":"/tmp/hello.txt","content":"Hello"}'
```

`path` 必填；`content` 最大 4MB、请求体最大 32MB，超出返回 413。写入会覆盖原文件，权限重置为 `0644`。

### 下载文件

```bash
GET /api/docker/container/<ID>/file/download?path=/etc/hosts&token=<JWT>
```

### 上传文件

```bash
isrvd_upload "/docker/container/<ID>/file/upload?path=/tmp" "file" "./local.txt"
```

- 目标目录 `path` 必须通过 **URL Query** 传递（不能放在表单字段中），为空返回 400
- 可重复提交多个 `file` 字段；可选表单字段 `relativePath` 与 `file` 按顺序对应，用于指定子路径，父目录自动创建
- 上传文件权限为 `0644`
- 请求总大小受 `server.maxUploadSize` 限制，超出返回 400（解析上传表单失败）

### 删除文件或目录

```bash
isrvd_delete "/docker/container/<ID>/file/rm?path=/tmp/old&recursive=true"
```

`path` 必填；`recursive` 默认 `false`（执行 `rm -f`，无法删除目录），删除目录需传 `recursive=true`（执行 `rm -rf`）。

### 创建目录

```bash
isrvd_post "/docker/container/<ID>/file/mkdir" '{"path":"/tmp/newdir"}'
```

执行 `mkdir -p`，递归创建，目录已存在时不报错。

### 重命名 / 移动

```bash
isrvd_post "/docker/container/<ID>/file/rename" '{"oldPath":"/tmp/a","newPath":"/tmp/b"}'
```

### 修改权限

```bash
isrvd_post "/docker/container/<ID>/file/chmod" '{"path":"/tmp/script.sh","mode":"755"}'
```
