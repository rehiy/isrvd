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

后台协程按 `monitor.interval` 配置的间隔采集主机和所有运行中容器的监控数据，写入 `{RootDirectory}/monitor/` 目录（NDJSON 格式，按天分文件，如 `host_2026-05-28.ndjson`，保留 **3 天**，每日凌晨自动清理）。

`monitor.interval` 合法值：`5`、`15`、`30`、`60`，单位为秒；其他值（含不填）均视为禁用。

| 参数 | 类型 | 说明 |
|------|------|------|
| type | string | `host`（默认）或 `container` |
| id | string | 容器 ID（type=container 时必填） |
| since | number | 时间窗口（秒），默认 3600；传 `0` 为实时模式（见下） |

响应为记录数组，每条记录格式：`{ ts: number, data: HostStat }`（host）或 `{ ts: number, data: ContainerStats }`（container）。

### 实时模式（since=0）

```bash
isrvd_get "/overview/monitor?since=0"
isrvd_get "/overview/monitor?type=container&id=<CONTAINER_ID>&since=0"
```

传 `since=0` 时返回单条记录（非数组）：

1. 优先返回最近 **30s** 内已有的最新一条采集数据
2. 若无缓存数据（采集禁用或刚启动），则主动实时采集一次，**不写入文件**

响应格式：`{ ts: number, data: HostStat }` 或 `null`（采集失败时）。
