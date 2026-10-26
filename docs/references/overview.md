# Overview（概览与监控）API

## 启动聚合（Bootstrap）

```bash
isrvd_get "/overview/bootstrap"
```

前端启动专用接口，**无需登录即可调用**（`AccessAnon`）。始终返回 `auth`；登录后另含 `probe`、`config`。

| 字段 | 类型 | 说明 |
|------|------|------|
| `auth` | object | 当前认证信息（`mode`（`jwt` / `header`）、`username`、`member`、`oidcEnabled`、`oidcBtnLabel`、`passkeyEnabled`、`passwordDisabled`、`passwordMinLength`）；未登录时省略 `username`/`member`；代理 Header 模式下 `oidcEnabled`、`passwordDisabled` 恒为 `false` |
| `probe` | object | 服务可用性，仅已登录时返回：`copilot`、`apisix`、`caddy`、`docker`、`swarm`、`compose`、`node`；各项并发探活，整体 5 秒超时；`copilot` 需 `copilot.baseUrl` 与 `apiKey` 均已配置才为 `true`；`node` 仅在 `center` 模式注册，表示多节点管理可用 |
| `config` | object | 前端启动所需的最小配置，仅已登录时返回：`title`、`maxUploadSize`、`marketplaceUrl`、`openapiEnabled`、`links` |

`config.title` 为系统配置中的浏览器标题，留空时前端使用 `iSrvd`；节点视角使用中控的标题。

---

## 版本信息

```bash
isrvd_get "/overview/version"
```

获取当前版本及最新版本信息，需要 `GET /api/overview/version` 路由权限（`AccessPerm`，创始人不受限）。

| 字段 | 类型 | 说明 |
|------|------|------|
| `current` | string | 当前运行版本 |
| `latest` | string | 远端最新版本（1 小时缓存） |
| `release` | string | 最新版 Release URL |
| `hasUpdate` | boolean | 是否有可用更新 |
| `updaterImage` | string | 推荐的 docker-updater 镜像，按国家代码自动选择（6 小时缓存）：探测到 `country_code=CN` 时返回 `docker.cnb.cool/rehiy/docker-updater:latest`，否则返回 `rehiy/docker-updater:latest`。探测失败时回退为默认镜像 |

---

## 监控数据

```bash
# 主机监控（实时模式）
isrvd_get "/overview/monitor?type=host&since=0"

# 主机监控（历史数据）
isrvd_get "/overview/monitor?type=host&since=3600"   # 最近1小时
isrvd_get "/overview/monitor?type=host&since=21600"  # 最近6小时
isrvd_get "/overview/monitor?type=host&since=43200"  # 最近12小时
isrvd_get "/overview/monitor?type=host&since=86400"  # 最近24小时

# 容器监控
isrvd_get "/overview/monitor?type=container&since=3600&id=<CONTAINER_ID>"
```

前端 `/local/monitor` 入口按 `GET /api/overview/monitor` 权限显示。

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | | `host`（默认）或 `container`；非 `container` 的值一律按 `host` 处理 |
| `since` | number | | 时间范围（秒），默认 `3600`：`0`=实时模式，其他任意正整数为历史窗口（常用 `3600`/`21600`/`43200`/`86400`）；非数字或负数按 `3600` 处理 |
| `id` | string | 条件 | `type=container` 时必填（缺失返回 400），容器 ID |

**记录格式：**

历史查询（`since>0`）返回记录数组，服务端会按请求的 `since` 时间窗口降采样，返回点数控制在约 **300** 个以内；历史数据保留 3 天。实时模式（`since=0`）返回单条记录且不写入文件；实时查询容器时 Docker 不可用或采集失败，响应 `payload` 为 `null`（`*Record` 空指针装入 `any` 后不会被 `omitempty` 省略）。采集间隔未配置（`monitor.interval` 禁用）时历史查询只返回已有数据，实时模式不受影响。

| 字段 | 类型 | 说明 |
|------|------|------|
| ts | number | 采集时间戳（Unix 秒） |
| data | object | host 为 `HostStat`；container 为 Docker 原生 `container.StatsResponse`（`/containers/<id>/stats` 原始结构） |
| container_id | string | 仅容器历史记录返回 |

**`HostStat`（host 的 `data`）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| system | object | 系统资源明细：`hostId`、`hostName`、`os`、`platform`、`kernelArch`、`createAt`、`uptime`、`cpuCore`、`cpuCoreLogic`、`cpuModel[]`、`cpuPercent[]`、`memoryTotal`、`memoryUsed`、`swapTotal`、`swapUsed`、`diskTotal`、`diskUsed`、`diskReadBytes`、`diskWriteBytes`、`diskPartition[]{device, mountpoint, fstype, total, used}`、`netBytesRecv`、`netBytesSent`、`netInterface[]{name, bytesRecv, bytesSent, dropin, dropout, ipv4List, ipv6List}`、`publicIpv4`、`publicIpv6` 等（字节单位；分区与汇总值已过滤虚拟挂载） |
| time | string | 系统当前本地时间，格式 `YYYY-MM-DD HH:mm:ss` |
| timezone | string | 系统当前时区，包含 UTC 偏移，如 `CST UTC+08:00` 或 `UTC+00:00` |
| diskIO | object[] | `{name, readBytes, writeBytes, readCount, writeCount}`（累计值） |
| gpu | object[] \| null | `{index, deviceKey, name, vendor, memoryUsed, memoryTotal, utilization, temperature, powerUsage, fanSpeed}`；无 GPU 时为 `null` |
| go | object | Go 运行态：`version`、`numCPU`、`numGoroutine` 及内存统计 |

---

## 升级程序

```bash
isrvd_post "/overview/upgrade"
```

触发 isrvd 程序自升级至最新版本：从 `https://isrvd.rehiy.com/update/` 下载 tar.gz 并原地替换二进制，成功返回 200 后约 500ms 重启。需要 `POST /api/overview/upgrade` 路由权限；已是最新版本返回 400"当前已是最新版本"。

Docker 容器部署场景下，前端使用 `GET /api/overview/version` 返回的 `updaterImage`，通过临时 docker-updater 容器拉取最新镜像并重启当前容器。

> ⚠️ 升级操作会重启服务，请确保已保存所有工作。

---

## 前端路由

| 路径 | 说明 |
|------|------|
| `/overview` | 概览页面（系统资源监控图表） |
| `/local/monitor` | 详细监控页面（时间范围选择：5分钟/1小时/6小时/12小时/24小时；5分钟模式加载一次历史后使用 `since=0` 实时单点轮询） |
