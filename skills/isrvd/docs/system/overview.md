# 系统概览 API

## 服务可用性探测

```bash
isrvd_get "/overview/probe"
```

| 字段 | 类型 | 说明 |
|------|------|------|
| agent | bool | Agent 是否配置 |
| apisix | bool | Apisix 是否可用 |
| caddy | bool | Caddy 是否可用 |
| docker | bool | Docker 是否可用 |
| swarm | bool | Swarm 是否可用 |
| compose | bool | Compose 是否可用 |
| versionCheck | object | `{latest, update, release}`，版本检测结果 |

## 监控历史数据

```bash
isrvd_get "/overview/monitor?type=host&since=3600"
isrvd_get "/overview/monitor?type=container&id=<CONTAINER_ID>&since=3600"
```

后台协程每 **15 秒**采集一次主机和所有运行中容器的监控数据，写入 `{RootDirectory}/monitor/` 目录（NDJSON 格式，按天分文件，如 `host_2026-05-28.ndjson`，保留 **3 天**，每日凌晨自动清理）。

| 参数 | 类型 | 说明 |
|------|------|------|
| type | string | `host`（默认）或 `container` |
| id | string | 容器 ID（type=container 时必填） |
| since | number | 时间窗口（秒），默认 3600 |

响应为记录数组，每条记录格式：`{ ts: number, data: HostStat }`（host）或 `{ ts: number, data: ContainerStats }`（container）。

> **获取实时数据**：取数组最后一条记录即为最新采集值，无需单独的 `/overview/status` 接口。
