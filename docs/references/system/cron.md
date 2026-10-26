# 计划任务 API

模块名：`cron`。除 `GET /api/cron/types` 仅需登录外，其余路由需在成员 `permissions` 中逐条授予对应的 `METHOD /api/cron/...` 路由权限（创始人除外）。

保存前会检查持久化内容是否与加载基线一致，变化时拒绝覆盖，需重载服务后重试，详见[保存一致性](config.md#保存一致性)。

---

## 数据结构

### CronJob

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 唯一标识（UUID，只读） |
| `name` | string | 任务名称 |
| `schedule` | string | Cron 表达式（5字段标准格式，如 `0 2 * * *`） |
| `type` | string | 脚本类型；以 `/api/cron/types` 返回为准；Docker 可用时额外包含 `DOCKER_TMP`（临时容器）与 `DOCKER_CTR`（exec 进现有容器） |
| `content` | string | 脚本内容。`SHELL`：以 `#!/` 开头按脚本文件执行，否则 `sh -c`；`EXEC`：按空白切分为命令与参数，不支持引号和重定向；Docker 类型使用 `/bin/sh` |
| `workDir` | string | 工作目录（绝对路径；Docker 类型无效） |
| `container` | string? | 目标容器名（`DOCKER_CTR` 必填） |
| `image` | string? | 镜像名（`DOCKER_TMP` 必填，创建临时容器运行脚本） |
| `volumes` | string? | 额外挂载（仅 `DOCKER_TMP`，换行分隔，格式 `/host:/container[:ro]`） |
| `timeout` | number | 超时秒数，0 表示不限制 |
| `enabled` | boolean | 是否启用（存储字段） |
| `registered` | boolean | 当前是否已注册到内存调度器（运行时字段，只读） |
| `entryId` | number? | robfig/cron 调度 entry ID（运行时字段，只读） |
| `runtimeStatus` | string | 当前调度状态：`scheduled` / `disabled` / `unregistered`（运行时字段，只读） |
| `description` | string | 描述（可选） |
| `nextRun` | string? | 下次预计执行时间（RFC3339，运行时字段，只读，仅已注册任务存在） |
| `lastRun` | string? | 上次计划调度时间（RFC3339，运行时字段，只读） |

计划任务配置存储在 `cron.yml`（位置见[业务数据存储](config.md#业务数据存储)）；`registered`、`entryId`、`runtimeStatus`、`nextRun`、`lastRun` 来自内存调度器，不写入存储。执行历史所有任务合并写入按天滚动的 JSONL 文件 `server.rootDirectory/cron/YYYY-MM-DD.jsonl`，每次执行追加一行（含 `jobId` 用于过滤）；保留最近 3 天，过期文件每日凌晨自动清理。

加载 `DOCKER` 类型任务时，按执行目标规范化类型：包含 `container` 的使用 `DOCKER_CTR`，包含 `image` 的使用 `DOCKER_TMP`。

Docker 不可用时，`/api/cron/types` 不返回 Docker 类型；已有的 `DOCKER_TMP` / `DOCKER_CTR` 任务加载时校验失败会被跳过：不调度，不出现在 `GET /cron/jobs` 列表中（删除、立即执行返回 404，更新、启停返回 400），但仍保留在 `cron.yml` 中，Docker 恢复并重载后重新生效。

### JobLog

| 字段 | 类型 | 说明 |
|------|------|------|
| `runId` | string | 单次执行 ID |
| `jobId` | string | 任务 ID |
| `jobName` | string | 任务名称 |
| `startTime` | string | 开始时间（RFC3339） |
| `endTime` | string | 结束时间（RFC3339） |
| `duration` | number | 执行耗时（毫秒） |
| `success` | boolean | 是否成功 |
| `output` | string | 标准输出与标准错误合并记录；最多保存 4 MiB，超出截断并追加 `[output truncated]` |
| `error` | string? | 错误信息（失败时存在） |

---

## 路由列表

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/cron/types` | 获取当前服务器可用脚本类型 |
| GET | `/api/cron/jobs` | 列出所有计划任务 |
| POST | `/api/cron/jobs` | 创建计划任务 |
| PUT | `/api/cron/jobs/:id` | 更新计划任务 |
| DELETE | `/api/cron/jobs/:id` | 删除计划任务 |
| POST | `/api/cron/jobs/:id/run` | 立即触发一次执行（异步） |
| PATCH | `/api/cron/jobs/:id` | 启用或禁用任务（部分更新 `enabled` 字段） |
| GET | `/api/cron/jobs/:id/logs` | 查询执行历史 |

---

## 接口详情

### 获取可用脚本类型

**GET** `/api/cron/types`

响应 `payload`：

```json
{
  "types": [
    { "value": "SHELL", "label": "Shell 脚本" },
    { "value": "EXEC", "label": "可执行文件" },
    { "value": "DOCKER_TMP", "label": "Docker 临时容器" },
    { "value": "DOCKER_CTR", "label": "Docker 现有容器" }
  ]
}
```

说明：Linux/macOS 返回 `SHELL`、`EXEC`；Windows 返回 `BAT`（BAT 批处理脚本）、`POWERSHELL`（PowerShell 脚本）、`EXEC`。仅当 Docker 可用时额外返回 `DOCKER_TMP`（需指定 `image`，可通过 `volumes` 挂载宿主机目录）与 `DOCKER_CTR`（需指定 `container`）。

---

### 列出所有任务

**GET** `/api/cron/jobs`

响应 `payload`：

```json
{
  "jobs": [ <CronJob>, ... ]
}
```

---

### 创建任务

**POST** `/api/cron/jobs`

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✓ | 任务名称 |
| `schedule` | string | ✓ | Cron 表达式 |
| `type` | string | ✓ | 取值以 `GET /api/cron/types` 返回为准 |
| `content` | string | ✓ | 脚本内容 |
| `workDir` | string | - | 工作目录；留空为 `server.rootDirectory`，相对路径基于 `rootDirectory` 解析，响应返回绝对路径（Docker 类型无效） |
| `image` | string | `DOCKER_TMP` 必填 | 临时容器镜像名 |
| `container` | string | `DOCKER_CTR` 必填 | 目标容器名，如 `my-python` |
| `volumes` | string | - | 仅 `DOCKER_TMP`：换行分隔，`/host:/container[:ro]` |
| `timeout` | number | - | 超时秒数（默认 0） |
| `enabled` | boolean | - | 创建后是否启用（默认 false） |
| `description` | string | - | 描述 |

响应 `payload`：`{ "job": <Job> }`，仅含存储字段，不含 `registered`、`entryId`、`runtimeStatus`、`nextRun`、`lastRun`；运行时状态请通过 `GET /cron/jobs` 查看。

---

### 更新任务

**PUT** `/api/cron/jobs/:id`

请求体同创建，为整体覆盖：未传的 `enabled`、`timeout` 等按零值处理（漏传 `enabled` 会禁用任务）。响应 `payload`：`{ "job": <Job> }`（仅存储字段）

---

### 删除任务

**DELETE** `/api/cron/jobs/:id`

无请求体，删除成功返回 200；任务不存在、调度器已停止或保存失败均返回 404。

---

### 立即执行

**POST** `/api/cron/jobs/:id/run`

无请求体。任务在后台异步执行，立即返回 200，执行结果记录到日志。

- 同一任务正在执行时返回 `409`
- 任务不存在或调度器已停止返回 `404`
- 定时触发时若上一次仍在执行，跳过本次

---

### 启用/禁用

**PATCH** `/api/cron/jobs/:id`

请求体仅支持 `enabled`（boolean）；不传按 `false` 处理，即禁用任务：

```json
{ "enabled": true }
```

---

### 查询执行历史

**GET** `/api/cron/jobs/:id/logs?limit=50`

Query 参数：

| 参数 | 说明 |
|------|------|
| `limit` | 返回最近 N 条，默认 50；取值 1–100，≤0 或 >100 时按 50 处理；非整数返回 400 |

响应 `payload`：

```json
{
  "logs": [ <JobLog>, ... ]
}
```

日志从 `server.rootDirectory/cron/YYYY-MM-DD.jsonl` 中按 `jobId` 过滤读取（从最近的当日文件向前回扫至多 3 天），按时间倒序排列（最新的在前）。

---

## Shell 示例

```bash
# 列出所有任务
isrvd_get "/cron/jobs"

# 创建每天 2 点执行的任务
isrvd_post "/cron/jobs" '{
  "name": "每日备份",
  "schedule": "0 2 * * *",
  "type": "SHELL",
  "content": "#!/bin/bash\ncd /data && tar czf backup-$(date +%Y%m%d).tar.gz .",
  "timeout": 300,
  "enabled": true
}'

# 立即触发
isrvd_post "/cron/jobs/<JOB_ID>/run" '{}'

# 禁用任务
isrvd_patch "/cron/jobs/<JOB_ID>" '{"enabled": false}'

# 查看最近 20 条执行日志
isrvd_get "/cron/jobs/<JOB_ID>/logs?limit=20"

# 删除任务
isrvd_delete "/cron/jobs/<JOB_ID>"
```

## 失败通知

在「系统配置 → 监控告警」开启计划任务失败通知，或按 [系统配置文档](config.md#应用故障告警) 更新 `notify.events.cronEnabled`。需要至少一个可用 Webhook 通道。

每次手动或定时执行失败（包括超时）会发送一次 `cron.failed`；事件包含 `jobId`、`jobName`、`runId` 和 `duration`（毫秒），不包含脚本、输出或原始错误。成功执行不推送；失败原因通过执行历史查看：

```bash
isrvd_get "/cron/jobs/<id>/logs?limit=20"
```

配置重载会停止旧调度器，防止重复调度；已接受的手动执行和已开始的定时任务继续执行，并按其调度器创建时的通知配置发送失败通知。旧实例停止后拒绝新增、修改、删除、启停和手动执行。

退出时的分段等待与取消规则见[应用故障告警](config.md#应用故障告警)；因进程退出而取消的任务不发送失败通知。
