# 计划任务 API

模块名：`cron`。除 `GET /api/cron/types` 仅需登录外，其余路由均需 `cron` 模块权限。

---

## 数据结构

### CronJob

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 唯一标识（UUID，只读） |
| `name` | string | 任务名称 |
| `schedule` | string | Cron 表达式（5字段标准格式，如 `0 2 * * *`） |
| `type` | string | 脚本类型；以服务器操作系统返回的 `/api/cron/types` 为准；所有系统均包含 `DOCKER` |
| `content` | string | 脚本内容 |
| `workDir` | string | 工作目录（可选，DOCKER 类型无效） |
| `container` | string? | 目标容器名（DOCKER 类型，exec 进现有容器；与 `image` 二选一） |
| `image` | string? | 镜像名（DOCKER 类型，创建临时容器运行脚本；与 `container` 二选一） |
| `volumes` | string? | 额外挂载（DOCKER + image 临时容器模式，换行分隔，格式 `/host:/container[:ro]`） |
| `timeout` | number | 超时秒数，0 表示不限制 |
| `enabled` | boolean | 是否启用（存储字段） |
| `registered` | boolean | 当前是否已注册到内存调度器（运行时字段，只读） |
| `entryId` | number? | robfig/cron 调度 entry ID（运行时字段，只读） |
| `runtimeStatus` | string | 当前调度状态：`scheduled` / `disabled` / `unregistered`（运行时字段，只读） |
| `description` | string | 描述（可选） |
| `nextRun` | string? | 下次预计执行时间（RFC3339，运行时字段，只读，仅已注册任务存在） |
| `lastRun` | string? | 上次计划调度时间（RFC3339，运行时字段，只读） |

计划任务配置存储在 `server.rootDirectory/cron.yml`，由服务自动读写；`registered`、`entryId`、`runtimeStatus`、`nextRun`、`lastRun` 来自当前内存调度器状态，不写入存储文件。执行历史按任务 ID 拆分为 JSONL 文件，写入 `server.rootDirectory/logs/cron/{jobId}.log`，每次执行追加一行结构化记录。

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
| `output` | string | 标准输出 |
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
| POST | `/api/cron/jobs/:id/enable` | 启用或禁用任务 |
| GET | `/api/cron/jobs/:id/logs` | 查询执行历史 |

---

## 接口详情

### 获取可用脚本类型

**GET** `/api/cron/types`

响应 `payload`：

```json
{
  "types": [
    { "value": "SHELL", "label": "SHELL（Shell 脚本）" },
    { "value": "EXEC", "label": "EXEC（直接执行命令）" }
  ]
}
```

说明：Linux/macOS 返回 `SHELL`、`EXEC`；Windows 返回 `BAT`、`POWERSHELL`、`EXEC`。所有平台都额外返回 `DOCKER`。`DOCKER` 类型需指定 `image`（临时容器）或 `container`（exec），二选一；`image` 模式下还可通过 `volumes` 挂载宿主机目录。

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
| `workDir` | string | - | 工作目录（DOCKER 类型无效） |
| `container` | string | DOCKER 类型必填 | 目标容器名，如 `my-python` |
| `timeout` | number | - | 超时秒数（默认 0） |
| `enabled` | boolean | - | 创建后是否启用（默认 false） |
| `description` | string | - | 描述 |

响应 `payload`：`{ "job": <CronJob> }`

---

### 更新任务

**PUT** `/api/cron/jobs/:id`

请求体同创建，响应 `payload`：`{ "job": <CronJob> }`

---

### 删除任务

**DELETE** `/api/cron/jobs/:id`

无请求体，删除成功返回 200。

---

### 立即执行

**POST** `/api/cron/jobs/:id/run`

无请求体。任务在后台异步执行，立即返回 200，执行结果记录到日志。

---

### 启用/禁用

**POST** `/api/cron/jobs/:id/enable`

请求体：

```json
{ "enabled": true }
```

---

### 查询执行历史

**GET** `/api/cron/jobs/:id/logs?limit=50`

Query 参数：

| 参数 | 说明 |
|------|------|
| `limit` | 返回最近 N 条，默认 50，最大 100 |

响应 `payload`：

```json
{
  "logs": [ <JobLog>, ... ]
}
```

日志从 `server.rootDirectory/logs/cron/{jobId}.log` 读取，按时间倒序排列（最新的在前）。

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
isrvd_post "/cron/jobs/<JOB_ID>/enable" '{"enabled": false}'

# 查看最近 20 条执行日志
isrvd_get "/cron/jobs/<JOB_ID>/logs?limit=20"

# 删除任务
isrvd_delete "/cron/jobs/<JOB_ID>"
```
