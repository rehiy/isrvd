# Overview（概览与监控）API

## 服务探测

```bash
isrvd_get "/overview/probe"
```

返回各服务的可用性状态：

| 字段 | 类型 | 说明 |
|------|------|------|
| Docker | boolean | Docker 服务是否可用 |
| Swarm | boolean | Swarm 服务是否可用 |
| Compose | boolean | Compose 服务是否可用 |
| Apisix | boolean | APISIX 服务是否可用 |
| Caddy | boolean | Caddy 服务是否可用 |

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

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | ✅ | `host` 或 `container` |
| `since` | number | ✅ | 时间范围（秒）：`0`=实时模式，`3600`=1小时，`21600`=6小时，`43200`=12小时，`86400`=24小时 |
| `id` | string | 条件 | `type=container` 时必填，容器 ID |

**响应字段（host）：**

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

触发 isrvd 程序自升级至最新版本（通过 `upgrade` 包实现）。

> ⚠️ 升级操作会重启服务，请确保已保存所有工作。

---

## 前端路由

| 路径 | 说明 |
|------|------|
| `/overview` | 概览页面（系统资源监控图表） |
| `/local/monitor` | 详细监控页面（时间范围选择：5分钟/1小时/6小时/12小时/24小时；5分钟模式加载一次历史后使用 `since=0` 实时单点轮询） |
