# Overview（概览与监控）API

## 启动聚合（Bootstrap）

```bash
isrvd_get "/overview/bootstrap"
```

前端启动专用接口，**无需登录即可调用**（`AccessAnon`）。一次返回 auth、probe、config 三段数据，替代原来串行的三次请求。

| 字段 | 类型 | 说明 |
|------|------|------|
| `auth` | object | 当前认证信息（`mode`、`username`、`member`、`oidcEnabled`、`oidcBtnLabel`、`passkeyEnabled`） |
| `probe` | object \| null | 服务可用性，已登录时返回（`agent`、`apisix`、`caddy`、`docker`、`swarm`、`compose`） |
| `config` | object \| null | 前端启动所需的最小配置，已登录时返回（`maxUploadSize`、`marketplaceUrl`、`openapiEnabled`、`links`） |

---

## 版本信息

```bash
isrvd_get "/overview/version"
```

获取当前版本及最新版本信息，登录即可访问（`AccessAuth`）。

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
| `type` | string | ✅ | `host` 或 `container` |
| `since` | number | ✅ | 时间范围（秒）：`0`=实时模式，`3600`=1小时，`21600`=6小时，`43200`=12小时，`86400`=24小时 |
| `id` | string | 条件 | `type=container` 时必填，容器 ID |

**响应字段（host）：**

历史查询（`since>0`）返回记录数组，服务端会按请求的 `since` 时间窗口降采样，返回点数控制在约 **300** 个以内；实时模式（`since=0`）返回单条记录且不写入文件。

记录格式：`{ ts: number, data: HostStat }`；下表为 `data` 中的主机监控字段。

| 字段 | 类型 | 说明 |
|------|------|------|
| timestamp | number | 时间戳（Unix 秒） |
| cpu_percent | number | CPU 使用率（百分比） |
| mem_percent | number | 内存使用率（百分比） |
| mem_used | number | 已用内存（字节） |
| mem_total | number | 总内存（字节） |
| load1 | number | 1分钟负载 |
| load5 | number | 5分钟负载 |
| load15 | number | 15分钟负载 |
| disk_root_percent | number | 根分区使用率（百分比） |
| disk_root_used | number | 根分区已用空间（字节） |
| disk_root_total | number | 根分区总空间（字节） |

**响应字段（container）：**

记录格式：`{ ts: number, data: ContainerStats }`；下表为 `data` 中的容器监控字段。

| 字段 | 类型 | 说明 |
|------|------|------|
| timestamp | number | 时间戳（Unix 秒） |
| cpu_percent | number | CPU 使用率（百分比） |
| mem_percent | number | 内存使用率（百分比） |
| mem_used | number | 已用内存（字节） |

---

## 升级程序

```bash
isrvd_post "/overview/upgrade"
```

触发 isrvd 程序自升级至最新版本（通过 `upgrade` 包实现，二进制原地替换并重启）。

Docker 容器部署场景下，前端改用 `GET /api/overview/version` 返回的 `updaterImage` 字段，通过临时 docker-updater 容器拉取最新镜像并重启当前容器；该镜像按国家代码自动选择（CN 使用 CNB 镜像源）。

> ⚠️ 升级操作会重启服务，请确保已保存所有工作。

---

## 前端路由

| 路径 | 说明 |
|------|------|
| `/overview` | 概览页面（系统资源监控图表） |
| `/local/monitor` | 详细监控页面（时间范围选择：5分钟/1小时/6小时/12小时/24小时；5分钟模式加载一次历史后使用 `since=0` 实时单点轮询） |
