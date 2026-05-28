# 系统概览 API

## 服务可用性探测

```bash
isrvd_get "/overview/probe"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| agent | object | `{available: bool}` |
| apisix | object | `{available: bool}` |
| caddy | object | `{available: bool}` |
| docker | object | `{available: bool}` |
| swarm | object | `{available: bool}` |
| compose | object | `{available: bool}` |

## 系统资源统计

```bash
isrvd_get "/overview/status"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| system | object | CPU、内存、磁盘、网络、uptime |
| diskIO | object[] | `{Name, ReadBytes, WriteBytes}` |
| gpu | object[] | `{name, vendor, memoryUsed, memoryTotal, utilization, temperature}` |
| go | object | `{version, numCPU, numGoroutine, alloc, heapAlloc, heapInuse, heapSys, stackInuse, stackSys, sys, totalAlloc, numGC, lastGC}` |
| version | string | isrvd 版本 |

## 监控历史数据

```bash
isrvd_get "/overview/history?type=host&since=3600"
isrvd_get "/overview/history?type=container&id=<CONTAINER_ID>&since=3600"
```

后台协程每 **15 秒**采集一次主机和所有运行中容器的监控数据，写入 `{RootDirectory}/monitor/` 目录（NDJSON 格式，按天分文件，如 `host_2026-05-28.ndjson`，保留 **3 天**，每日凌晨自动清理）。

| 参数 | 类型 | 说明 |
|------|------|------|
| type | string | `host`（默认）或 `container` |
| id | string | 容器 ID（type=container 时必填） |
| since | number | 时间窗口（秒），默认 3600 |

响应为记录数组，每条记录格式：`{ ts: number, data: SystemStat }`（host）或 `{ ts: number, data: ContainerStats }`（container）。
