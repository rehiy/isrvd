# 本机进程管理 API

## 概述

Local 模块提供 isrvd 所在主机的进程查看与终止能力，进程列表按常驻内存降序返回，便于快速定位占用最高的进程。

---

## 查询进程列表

```
GET /local/processes
```

**响应字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `pid` | number | 进程 ID |
| `ppid` | number | 父进程 ID |
| `name` | string | 进程名 |
| `username` | string | 运行用户 |
| `status` | string | 运行状态 |
| `cpuPercent` | number | CPU 占用（%，自启动以来的均值，多核可超过 100） |
| `memoryPercent` | number | 内存占用（%） |
| `memoryRss` | number | 常驻内存（字节） |
| `createTime` | number | 启动时间（Unix 毫秒） |
| `cmdline` | string | 完整命令行，仅创始人返回；普通成员不返回该字段以避免命令行凭据泄露 |

单个进程的某个字段采集失败时降级为零值，不会导致该进程从列表中消失。

```bash
isrvd_get "/local/processes"
```

---

## 终止进程

```
POST /local/process/:pid/kill
```

**请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `force` | boolean | 否 | `true` 发送 SIGKILL，否则发送 SIGTERM（默认） |

```bash
# 优雅终止
isrvd_post "/local/process/1234/kill" '{"force": false}'

# 强制终止
isrvd_post "/local/process/1234/kill" '{"force": true}'
```

**保护规则：**

- PID ≤ 1 的进程不可终止
- isrvd 自身进程不可终止（避免面板自杀）
- 命中保护时返回 403

接口只保证信号送达，进程是否退出取决于其对信号的处理；建议终止后复查列表确认。

---

## 权限要求

- 查询（`GET /api/local/processes`）：需要对应路由权限
- 查询（`GET /api/local/processes`）：需要对应路由权限；完整命令行仅创始人可见
- 终止（`POST /api/local/process/:pid/kill`）：除对应路由权限外，**仅创始人可执行**，且强制记入审计日志

---

## 说明

- CPU 占用为自启动以来的均值，与 `ps` 的 `%CPU` 口径一致，不是瞬时采样值
- 列表为调用时刻的快照，高频创建退出的短命进程可能未被采集
- 终止高危，操作前请确认进程归属
