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
| `config` | object | 前端启动所需的最小配置，仅已登录时返回：`title`、`maxUploadSize`、`maxEditSize`（在线编辑文件大小上限，字节）、`marketplaceUrl`、`openapiEnabled`、`links` |

`config.title` 为系统配置中的浏览器标题，留空时前端使用 `iSrvd`；节点视角下 `title` 取自节点，`links` 为中控与节点的合并（中控在前、按 `url` 去重，仅保留 `http(s)` 链接）。

---

## 版本信息

```bash
isrvd_get "/overview/version"
```

获取当前版本及最新版本信息，需要 `GET /api/overview/version` 路由权限（`AccessPerm`，创始人不受限）。`hasUpdate` 直接使用 libgo 检查结果：仅升级服务器提供可用包且没有业务错误时为 `true`，不再自行比较版本号。

升级轮询使用 `/overview/version?current=1`，仅读取当前运行版本，不触发远端版本检查或镜像源探测；响应结构不变，其他字段返回零值。

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

原接口和 `POST /api/overview/upgrade` 权限不变。提交后立即返回 HTTP 200（`payload` 为空），仅表示已受理；后台检查、下载、校验并替换二进制，成功后约500ms重启。重复提交不会启动第二次升级，失败详情记录在升级日志中，不提供任务ID或状态查询。

使用正式依赖 libgo v0.22.0。版本检查限时30秒，检查及下载共用4分30秒期限；升级器初始化时绑定上下文，后续 `Check()`、`Apply()` 直接复用，超时实际取消请求并清理暂存文件，后台结束后可重试。isrvd 从绑定上下文的响应流直接提取 tar.gz 中的二进制，不再保存中间归档；下载及解压读取响应取消，总解压量限制为512 MiB。检查、HTTP 请求、落盘清理、替换及回滚复用 libgo；服务端提供 `checksum` 时校验解压后二进制的 SHA256，未提供时跳过校验。版本检查与下载共享 HTTPS 和重定向域名限制。服务关闭时取消下载并等待升级工作退出，避免打断文件替换，且不再启动升级或重启。

二进制和容器升级共用提交与版本轮询流程，通过 `GET /api/overview/version?current=1` 的运行版本变化确认完成。二进制提交限时10秒，容器提交限时2分钟；提交后最多等待5分钟，每次查询限时10秒。两种提交遇到断连或不确定的网关502/503/504/524继续等待、不重发POST；提交返回业务503拒绝（`success=false`，如节点离线或服务停止）立即恢复按钮，不进入轮询。轮询中的暂时离线仍继续等待，401/403停止等待。离开页面取消提交请求及轮询，不再触发后续刷新，但不撤销后台已经受理的升级。版本查询及镜像源探测也使用请求上下文，避免前端取消后继续挂起。失败详情查看日志，前端最终提示等待超时；错误提示沿用原客户端行为，未增加Axios扩展。

Docker 容器升级仍通过临时 docker-updater 容器执行，不改变部署流程。

> ⚠️ 升级操作会重启服务，请确保已保存所有工作。

---

## 前端路由

| 路径 | 说明 |
|------|------|
| `/overview` | 概览页面（系统资源监控图表） |
| `/local/monitor` | 详细监控页面（时间范围选择：5分钟/1小时/6小时/12小时/24小时；5分钟模式加载一次历史后使用 `since=0` 实时单点轮询） |
